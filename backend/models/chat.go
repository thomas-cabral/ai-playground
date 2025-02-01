package models

import (
	"gorm.io/gorm"
)

type Chat struct {
	gorm.Model
	Messages []Message
}

type Message struct {
	gorm.Model
	ChatID  uint
	Role    string
	Content string
}
