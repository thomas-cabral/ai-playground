package main

import (
	"log"
	"web/ai-playground/controllers"
	"web/ai-playground/models"
	"web/ai-playground/services"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// Initialize SQLite database
	db, err := gorm.Open(sqlite.Open("chat.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}

	// Migrate the schema
	err = db.AutoMigrate(&models.Chat{}, &models.Message{})
	if err != nil {
		log.Fatal("failed to migrate database:", err)
	}

	// Initialize services with db
	openRouterService := services.NewOpenRouterService(db)

	// Initialize controllers
	chatController := controllers.NewChatController(openRouterService)

	// Set up Gin router
	router := gin.Default()

	// Configure CORS
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Routes
	setupRoutes(router, chatController)

	// Start server
	if err := router.Run(":8088"); err != nil {
		log.Fatal("failed to start server:", err)
	}
}

func setupRoutes(r *gin.Engine, cc *controllers.ChatController) {
	api := r.Group("/api")
	{
		api.POST("/chat", cc.HandleChat)
		api.GET("/chat", cc.HandleGetChats)
		api.POST("/chat/new", cc.HandleNewChat)
		api.GET("/chat/:id", cc.HandleGetChat)
		api.POST("/chat/:id/star", cc.HandleToggleChatStar)
		api.POST("/message/:id/star", cc.HandleToggleMessageStar)
		api.DELETE("/chat/:id", cc.HandleDeleteChat)
		api.POST("/chat/fork", cc.HandleForkChat)
		api.GET("/chat/:id/forks", cc.HandleGetChatForks)
		api.GET("/chat/:id/fork-message/:messageId", cc.HandleGetParentForkMessage)
	}
}
