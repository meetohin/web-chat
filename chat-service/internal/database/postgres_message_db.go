package database

import (
	"database/sql"
	_ "github.com/lib/pq"
	"time"
)

type PostgreSQLMessageRepository struct {
	db *sql.DB
}

func NewPostgreSQLMessageRepository(db *sql.DB) *PostgreSQLMessageRepository {
	return &PostgreSQLMessageRepository{db: db}
}

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

	// Разворачиваем, чтобы старые сообщения были вначале
	for i := 0; i < len(messages)/2; i++ {
		j := len(messages) - i - 1
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *PostgreSQLMessageRepository) GetMessageCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM messages").Scan(&count)
	return count, err
}

// Инициализация схемы БД
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
