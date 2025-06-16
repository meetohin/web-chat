package repository

import (
	"database/sql"
	"errors"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username string
	Password string // hashed
}

type PostgreSQLUserRepository struct {
	db *sql.DB
}

func NewPostgreSQLUserRepository(db *sql.DB) *PostgreSQLUserRepository {
	return &PostgreSQLUserRepository{db: db}
}

func (r *PostgreSQLUserRepository) CreateUser(username, password string) error {
	// Check if user already exists
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("user already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Insert user
	_, err = r.db.Exec(
		"INSERT INTO users (username, password, created_at) VALUES ($1, $2, NOW())",
		username, string(hashedPassword),
	)
	return err
}

func (r *PostgreSQLUserRepository) GetUser(username string) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(
		"SELECT username, password FROM users WHERE username = $1",
		username,
	).Scan(&user.Username, &user.Password)

	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *PostgreSQLUserRepository) ValidatePassword(username, password string) bool {
	user, err := r.GetUser(username)
	if err != nil {
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

// Initialize database schema
func (r *PostgreSQLUserRepository) CreateTables() error {
	query := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        username VARCHAR(50) UNIQUE NOT NULL,
        password VARCHAR(255) NOT NULL,
        created_at TIMESTAMP DEFAULT NOW()
    );
    CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
    `
	_, err := r.db.Exec(query)
	return err
}
