package database

import (
	"sync"
	"time"
)

type Message struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
}

type MessageRepository struct {
	messages []Message
	mu       sync.RWMutex
	nextID   int
}

func NewMessageRepository() *MessageRepository {
	return &MessageRepository{
		messages: make([]Message, 0),
		nextID:   1,
	}
}

func (r *MessageRepository) SaveMessage(username, text string) *Message {
	r.mu.Lock()
	defer r.mu.Unlock()

	message := Message{
		ID:        r.nextID,
		Username:  username,
		Text:      text,
		Timestamp: time.Now(),
	}

	r.messages = append(r.messages, message)
	r.nextID++

	return &message
}

func (r *MessageRepository) GetRecentMessages(limit int) []Message {
	r.mu.RLock()
	defer r.mu.RUnlock()

	start := len(r.messages) - limit
	if start < 0 {
		start = 0
	}

	return r.messages[start:]
}

func (r *MessageRepository) GetMessageCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.messages)
}
