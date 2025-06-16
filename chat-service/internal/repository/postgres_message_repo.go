package repository

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

// PostgreSQLMessageRepository implements MessageRepository interface
type PostgreSQLMessageRepository struct {
	db *sql.DB
}

// NewPostgreSQLMessageRepository creates a new PostgreSQL message repository
func NewPostgreSQLMessageRepository(db *sql.DB) MessageRepository {
	return &PostgreSQLMessageRepository{db: db}
}

// SaveMessage saves a new message to the database
func (r *PostgreSQLMessageRepository) SaveMessage(username, text string) (*Message, error) {
	message := &Message{
		Username:  username,
		Text:      text,
		Timestamp: time.Now(),
	}

	err := r.db.QueryRow(
		"INSERT INTO messages (username, text, created_at) VALUES ($1, $2, $3) RETURNING id",
		username, text, message.Timestamp,
	).Scan(&message.ID)

	if err != nil {
		return nil, err
	}

	return message, nil
}

// GetRecentMessages retrieves recent messages from the database
func (r *PostgreSQLMessageRepository) GetRecentMessages(limit int) ([]Message, error) {
	rows, err := r.db.Query(
		"SELECT id, username, text, created_at FROM messages ORDER BY created_at DESC LIMIT $1",
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.Username, &msg.Text, &msg.Timestamp)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}

	// Reverse to show older messages first
	for i := 0; i < len(messages)/2; i++ {
		j := len(messages) - i - 1
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetMessageCount returns the total number of messages
func (r *PostgreSQLMessageRepository) GetMessageCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	return count, err
}

// CreateTables initializes the database schema
func (r *PostgreSQLMessageRepository) CreateTables() error {
	query := `
    CREATE TABLE IF NOT EXISTS messages (
        id SERIAL PRIMARY KEY,
        username VARCHAR(50) NOT NULL,
        text TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT NOW()
    );
    CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at DESC);
    CREATE INDEX IF NOT EXISTS idx_messages_username ON messages(username);
    `
	_, err := r.db.Exec(query)
	return err
}
