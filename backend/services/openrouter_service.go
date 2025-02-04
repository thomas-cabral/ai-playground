package services

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"web/ai-playground/models"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
)

type OpenRouterService struct {
	APIKey  string
	BaseURL string
	DB      *gorm.DB
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	ChatID   uint      `json:"chat_id,omitempty"`
}

type Message struct {
	ID        uint   `json:"id,omitempty"`
	ChatID    uint   `json:"chat_id,omitempty"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	ModelName string `json:"model_name"`
}

type ChatResponse struct {
	ID      string    `json:"id"`
	Choices []Choice  `json:"choices"`
	Error   *APIError `json:"error,omitempty"`
}

type Choice struct {
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type APIError struct {
	Message string `json:"message"`
}

type StreamChoice struct {
	Index int `json:"index"`
	Delta struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type UsageData struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type StreamResponse struct {
	ID      string         `json:"id"`
	Choices []StreamChoice `json:"choices"`
	Usage   *UsageData     `json:"usage,omitempty"`
}

func NewOpenRouterService(db *gorm.DB) *OpenRouterService {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	openRouterAPIKey := os.Getenv("OPENROUTER_API_KEY")
	return &OpenRouterService{
		APIKey:  openRouterAPIKey,
		BaseURL: "https://openrouter.ai/api/v1",
		DB:      db,
	}
}

func (s *OpenRouterService) Chat(req ChatRequest, chatID uint, w http.ResponseWriter) error {
	var chat models.Chat

	// Load or create chat
	if chatID != 0 {
		if err := s.DB.First(&chat, chatID).Error; err != nil {
			return fmt.Errorf("error loading existing chat: %v", err)
		}
	} else {
		chat = models.Chat{}
		if err := s.DB.Create(&chat).Error; err != nil {
			return fmt.Errorf("error creating new chat: %v", err)
		}
		chatID = chat.ID
	}

	// Save new messages
	for _, msg := range req.Messages {
		// Skip messages that already exist in the database
		if msg.ID != 0 {
			continue
		}

		// Create new message
		message := models.Message{
			ChatID:    chatID,
			Role:      msg.Role,
			Content:   msg.Content,
			ModelName: req.Model,
		}
		if err := s.DB.Create(&message).Error; err != nil {
			return fmt.Errorf("error saving message: %v", err)
		}
	}

	// Remove the old chat ID check since we're handling it with the request parameter now
	url := fmt.Sprintf("%s/chat/completions", s.BaseURL)

	// Create a new request body with only the required fields for the API
	apiReq := struct {
		Model    string        `json:"model"`
		Messages []ChatMessage `json:"messages"`
		Stream   bool          `json:"stream"`
	}{
		Model:    req.Model,
		Messages: make([]ChatMessage, len(req.Messages)),
		Stream:   req.Stream,
	}

	// Convert messages to the format expected by the API
	for i, msg := range req.Messages {
		apiReq.Messages[i] = ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Add required headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.APIKey))
	httpReq.Header.Set("HTTP-Referer", "http://localhost:8080")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
			return fmt.Errorf("error response with status %d", resp.StatusCode)
		}
		return fmt.Errorf("API error: %s (code: %d)", errorResp.Error.Message, errorResp.Error.Code)
	}

	// Set up SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Create a buffered reader for the response body
	reader := bufio.NewReader(resp.Body)

	// After successful API response, save the assistant's response
	// This should be added before the streaming loop
	assistantMessage := models.Message{
		ChatID:  chat.ID,
		Role:    "assistant",
		Content: "", // This will be populated as we stream
	}
	if err := s.DB.Create(&assistantMessage).Error; err != nil {
		return fmt.Errorf("error saving assistant message: %v", err)
	}

	// Create a buffer to store the complete response
	var responseBuffer bytes.Buffer

	// Stream the response
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %v", err)
		}

		// Add to our buffer
		responseBuffer.Write(line)

		// Write each line as it comes in
		if _, err := w.Write(line); err != nil {
			return fmt.Errorf("error writing to response: %v", err)
		}

		// Flush the writer to send the chunk immediately
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}

	// Parse the accumulated response to extract the assistant's message
	var fullResponse string
	// Declare variables to store token usage if present in the stream
	var promptTokens, completionTokens, totalTokens int

	responseLines := bytes.Split(responseBuffer.Bytes(), []byte("\n"))
	fmt.Printf("Debug: Total response lines: %d\n", len(responseLines))

	for _, line := range responseLines {
		if len(line) == 0 {
			continue
		}
		if bytes.HasPrefix(line, []byte("data: ")) {
			data := bytes.TrimPrefix(line, []byte("data: "))
			fmt.Printf("Debug: Processing data line: %s\n", string(data))

			if string(data) == "[DONE]" {
				continue
			}

			var streamResponse StreamResponse
			if err := json.Unmarshal(data, &streamResponse); err != nil {
				fmt.Printf("Debug: JSON unmarshal error: %v\n", err)
				continue
			}

			// Check for the usage field in the stream response and capture it
			if streamResponse.Usage != nil {
				promptTokens = streamResponse.Usage.PromptTokens
				completionTokens = streamResponse.Usage.CompletionTokens
				totalTokens = streamResponse.Usage.TotalTokens
			}

			if len(streamResponse.Choices) > 0 {
				content := streamResponse.Choices[0].Delta.Content
				fmt.Printf("Debug: Found content: %s\n", content)
				fullResponse += content
			}
		}
	}

	fmt.Printf("Debug: Final full response: %s\n", fullResponse)

	// Update the assistant's message with the complete response
	if err := s.DB.Model(&assistantMessage).Update("content", fullResponse).Error; err != nil {
		return fmt.Errorf("error updating assistant message: %v", err)
	}

	// If token usage was extracted, update the assistant's message with the token usage data
	if promptTokens != 0 || completionTokens != 0 || totalTokens != 0 {
		updates := map[string]interface{}{
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      totalTokens,
		}
		if err := s.DB.Model(&assistantMessage).Updates(updates).Error; err != nil {
			return fmt.Errorf("error updating assistant message with usage: %v", err)
		}
	}

	return nil
}

// Add new method to get chat history
func (s *OpenRouterService) GetChatHistory(chatID uint) (*models.Chat, error) {
	var chat models.Chat
	if err := s.DB.Preload("Messages").First(&chat, chatID).Error; err != nil {
		return nil, fmt.Errorf("error fetching chat history: %v", err)
	}
	return &chat, nil
}
