package model

import (
	"errors"
	"time"
)

type Notification struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Title     string    `json:"title" db:"title"`
	Message   string    `json:"message" db:"message"`
	Type      string    `json:"type" db:"type"`
	IsRead    bool      `json:"is_read" db:"is_read"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type NotificationRequest struct {
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

func (n *NotificationRequest) Validate() error {
	if n.UserID == "" {
		return errors.New("user_id is required")
	}
	if n.Title == "" {
		return errors.New("title is required")
	}
	if n.Message == "" {
		return errors.New("message is required")
	}
	if n.Type == "" {
		n.Type = "message"
	}
	return nil
}

type WebSocketMessage struct {
	Type string       `json:"type"`
	Data Notification `json:"data"`
}

// Константы
const (
	TypeMessage       = "message"
	TypeMention       = "mention"
	TypeDirectMessage = "direct_message"
)
