package repository

// User represents a user in the system
type User struct {
	Username string
	Password string // hashed
}

// UserRepository defines the interface for user data access
type UserRepository interface {
	CreateUser(username, password string) error
	GetUser(username string) (*User, error)
	ValidatePassword(username, password string) bool
}
