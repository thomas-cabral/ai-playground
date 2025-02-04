package controllers

import (
	"fmt"
	"strconv"
	"web/ai-playground/models"
	"web/ai-playground/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ChatController struct {
	openRouterService *services.OpenRouterService
}

func NewChatController(openRouterService *services.OpenRouterService) *ChatController {
	return &ChatController{
		openRouterService: openRouterService,
	}
}

func (cc *ChatController) HandleChat(c *gin.Context) {
	var chatReq services.ChatRequest
	if err := c.ShouldBindJSON(&chatReq); err != nil {
		fmt.Printf("Error binding JSON: %v\n", err)
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Create or get existing chat
	var chat models.Chat
	isNewChat := true

	// If we have a chat ID in the request, try to find the existing chat
	if chatReq.ChatID != 0 {
		isNewChat = false
		if err := cc.openRouterService.DB.Preload("Messages").First(&chat, chatReq.ChatID).Error; err != nil {
			fmt.Printf("Error finding chat: %v\n", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
	}

	if isNewChat {
		chat = models.Chat{
			ModelName: chatReq.Model,
		}
		if err := cc.openRouterService.DB.Create(&chat).Error; err != nil {
			fmt.Printf("Error creating chat: %v\n", err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		// Set chat ID in response header for new chats
		c.Header("X-Chat-ID", fmt.Sprintf("%d", chat.ID))
	}

	fmt.Printf("Received request for model: %s\n", chatReq.Model)

	if chatReq.Stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
	}

	if err := cc.openRouterService.Chat(chatReq, chat.ID, c.Writer); err != nil {
		fmt.Printf("Error from OpenRouter service: %v\n", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
}

func (cc *ChatController) HandleGetChats(c *gin.Context) {
	// Parse pagination parameters
	page := c.DefaultQuery("page", "1")
	pageSize := 15 // Fixed page size

	pageNum, err := strconv.Atoi(page)
	if err != nil || pageNum < 1 {
		pageNum = 1
	}

	offset := (pageNum - 1) * pageSize

	var chats []models.Chat
	// Get the first message for preview and count all messages
	if err := cc.openRouterService.DB.
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Where("parent_id IS NULL").
		Order("created_at desc").
		Limit(pageSize).
		Offset(offset).
		Find(&chats).Error; err != nil {
		fmt.Printf("Error fetching chats: %v\n", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Get total count for pagination - only count root chats
	var total int64
	if err := cc.openRouterService.DB.Model(&models.Chat{}).Where("parent_id IS NULL").Count(&total).Error; err != nil {
		fmt.Printf("Error counting chats: %v\n", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Get message counts for each chat
	type ChatCount struct {
		ChatID uint
		Count  int64
	}
	var chatCounts []ChatCount
	if err := cc.openRouterService.DB.Model(&models.Message{}).
		Select("chat_id, count(*) as count").
		Group("chat_id").
		Find(&chatCounts).Error; err != nil {
		fmt.Printf("Error counting messages: %v\n", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// Create a map of chat ID to message count
	countMap := make(map[uint]int64)
	for _, cc := range chatCounts {
		countMap[cc.ChatID] = cc.Count
	}

	// Create response with additional fork information
	type ChatResponse struct {
		models.Chat
		MessageCount int64 `json:"messageCount"`
	}

	chatsWithCount := make([]ChatResponse, len(chats))
	for i, chat := range chats {
		chatsWithCount[i] = ChatResponse{
			Chat:         chat,
			MessageCount: countMap[chat.ID],
		}
	}

	hasMore := offset+len(chats) < int(total)

	c.JSON(200, gin.H{
		"chats":    chatsWithCount,
		"total":    total,
		"page":     pageNum,
		"pageSize": pageSize,
		"hasMore":  hasMore,
	})
}

func (cc *ChatController) HandleGetChat(c *gin.Context) {
	chatID := c.Param("id")
	var chat models.Chat

	if err := cc.openRouterService.DB.Preload("Messages").First(&chat, chatID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Chat not found"})
		return
	}

	c.JSON(200, chat)
}

func (cc *ChatController) HandleNewChat(c *gin.Context) {
	var req struct {
		Model string `json:"model"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	chat := models.Chat{
		ModelName: req.Model,
	}
	if err := cc.openRouterService.DB.Create(&chat).Error; err != nil {
		fmt.Printf("Error creating new chat: %v\n", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"id": chat.ID})
}

func (cc *ChatController) HandleToggleChatStar(c *gin.Context) {
	chatID := c.Param("id")
	var chat models.Chat

	if err := cc.openRouterService.DB.First(&chat, chatID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Chat not found"})
		return
	}

	// Toggle the starred status
	chat.Starred = !chat.Starred

	if err := cc.openRouterService.DB.Save(&chat).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"starred": chat.Starred})
}

func (cc *ChatController) HandleToggleMessageStar(c *gin.Context) {
	messageID := c.Param("id")
	var message models.Message

	if err := cc.openRouterService.DB.First(&message, messageID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Message not found"})
		return
	}

	// Toggle the starred status
	message.Starred = !message.Starred

	if err := cc.openRouterService.DB.Save(&message).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"starred": message.Starred})
}

func (cc *ChatController) HandleDeleteChat(c *gin.Context) {
	chatID := c.Param("id")

	// Soft delete the chat
	if err := cc.openRouterService.DB.Delete(&models.Chat{}, chatID).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"message": "Chat deleted successfully"})
}

func (cc *ChatController) HandleForkChat(c *gin.Context) {
	var req struct {
		ChatID     uint   `json:"chatId"`
		MessageID  uint   `json:"messageId"`
		NewContent string `json:"newContent"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Get the original chat
	var originalChat models.Chat
	if err := cc.openRouterService.DB.Preload("Messages", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	}).First(&originalChat, req.ChatID).Error; err != nil {
		c.JSON(404, gin.H{"error": "Original chat not found"})
		return
	}

	// Create new chat as a fork
	newChat := models.Chat{
		ModelName:     originalChat.ModelName,
		ParentID:      &originalChat.ID,
		ForkMessageID: &req.MessageID,
	}

	if err := cc.openRouterService.DB.Create(&newChat).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to create forked chat"})
		return
	}

	// Copy messages up to the fork point (but don't add the edited message)
	for _, msg := range originalChat.Messages {
		if msg.ID == req.MessageID {
			break
		}
		// Copy the message to the new chat
		newMsg := models.Message{
			ChatID:           newChat.ID,
			Role:             msg.Role,
			Content:          msg.Content,
			ModelName:        msg.ModelName,
			PromptTokens:     msg.PromptTokens,
			CompletionTokens: msg.CompletionTokens,
			TotalTokens:      msg.TotalTokens,
		}
		if err := cc.openRouterService.DB.Create(&newMsg).Error; err != nil {
			c.JSON(500, gin.H{"error": "Failed to copy message"})
			return
		}
	}

	// Send the new chat ID
	c.Header("X-Fork-Chat-ID", fmt.Sprintf("%d", newChat.ID))
	c.JSON(200, gin.H{"id": newChat.ID})
}

func (cc *ChatController) HandleGetChatForks(c *gin.Context) {
	chatID := c.Param("id")
	var forks []struct {
		MessageID uint `json:"messageId"`
		ForkID    uint `json:"forkId"`
	}

	if err := cc.openRouterService.DB.
		Table("chats").
		Select("fork_message_id as MessageID, id as ForkID").
		Where("parent_id = ?", chatID).
		Scan(&forks).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, forks)
}

func (cc *ChatController) HandleGetParentForkMessage(c *gin.Context) {
	messageID := c.Param("messageId")

	var message models.Message

	if err := cc.openRouterService.DB.
		Where("id = ?", messageID).
		Preload("Chat").
		First(&message).Error; err != nil {
		c.JSON(404, gin.H{"error": "Message not found"})
		return
	}

	c.JSON(200, gin.H{
		"id":             message.ID,
		"messageContent": message.Content,
		"chatId":         message.ChatID,
		"createdAt":      message.CreatedAt,
	})
}
