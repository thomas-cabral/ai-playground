package models

import (
	"gorm.io/gorm"
)

// Base model that properly exposes gorm.Model fields as JSON
type BaseModel struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type Chat struct {
	BaseModel
	Messages  []Message `json:"messages"`
	ModelName string    `json:"model_name"`
	Starred   bool      `json:"starred" gorm:"default:false"`
}

type Message struct {
	BaseModel
	ChatID    uint   `json:"chat_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	ModelName string `json:"model_name"`
	Starred   bool   `json:"starred" gorm:"default:false"`
}
