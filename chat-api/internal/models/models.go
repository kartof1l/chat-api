package models

import (
	"time"
)

type Chat struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"size:200;not null" json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Messages  []Message `gorm:"foreignKey:ChatID;constraint:OnDelete:CASCADE;" json:"messages,omitempty"`
}

type Message struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ChatID    uint      `gorm:"not null" json:"chat_id"`
	Text      string    `gorm:"size:5000;not null" json:"text"`
	CreatedAt time.Time `json:"created_at"`
}
