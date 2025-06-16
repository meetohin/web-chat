package repository

import (
	"database/sql"
	"errors"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// PostgreSQLUserRepository implements UserRepository interface
type PostgreSQLUserRepository struct {
	db *sql.DB
}

// NewPostgreSQLUserRepository creates a new PostgreSQL user repository
func NewPostgreSQLUserRepository(db *sql.DB) UserRepository {
	return &PostgreSQLUserRepository{db: db}
}

// CreateUser creates a new user in the repository
func (r *PostgreSQLUserRepository) CreateUser(username, password string) error {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(
		"INSERT INTO users (username, password, created_at) VALUES ($1, $2, NOW())",
		username, string(hashedPassword),
	)
	return err
}

// GetUser retrieves a user by username
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

// ValidatePassword validates a user's password
func (r *PostgreSQLUserRepository) ValidatePassword(username, password string) bool {
	user, err := r.GetUser(username)
	if err != nil {
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

// CreateTables initializes the repository schema
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
