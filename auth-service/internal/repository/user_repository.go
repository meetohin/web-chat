package repository

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
	"sync"
)

type User struct {
	Username string
	Password string // hashed
}

type UserRepository struct {
	users map[string]*User
	mu    sync.RWMutex
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: make(map[string]*User),
	}
}

func (r *UserRepository) CreateUser(username, password string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[username]; exists {
		return errors.New("user already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	r.users[username] = &User{
		Username: username,
		Password: string(hashedPassword),
	}

	return nil
}

func (r *UserRepository) GetUser(username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[username]
	if !exists {
		return nil, errors.New("user not found")
	}

	return user, nil
}

func (r *UserRepository) ValidatePassword(username, password string) bool {
	user, err := r.GetUser(username)
	if err != nil {
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}
