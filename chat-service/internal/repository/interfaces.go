package repository

import "time"

// Message represents a chat message
type Message struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

// MessageRepository defines the interface for message data access
type MessageRepository interface {
	SaveMessage(username, text string) (*Message, error)
	GetRecentMessages(limit int) ([]Message, error)
	GetMessageCount() (int, error)
}
