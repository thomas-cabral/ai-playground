package controllers

import (
	"fmt"
	"web/ai-playground/models"
	"web/ai-playground/services"

	"github.com/gin-gonic/gin"
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
		chat = models.Chat{}
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
	var chats []models.Chat
	if err := cc.openRouterService.DB.Preload("Messages").Find(&chats).Error; err != nil {
		fmt.Printf("Error fetching chats: %v\n", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, chats)
}

func (cc *ChatController) HandleNewChat(c *gin.Context) {
	// Create a new chat
	chat := models.Chat{}
	if err := cc.openRouterService.DB.Create(&chat).Error; err != nil {
		fmt.Printf("Error creating new chat: %v\n", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"id": chat.ID})
}
