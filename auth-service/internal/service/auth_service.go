package service

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/meetohin/web-chat/auth-service/internal/repository"
	"time"
)

var jwtSecret = []byte("your-secret-key") // TODO: env

type AuthService struct {
	userRepo *repository.UserRepository
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{
		userRepo: userRepo,
	}
}

func (s *AuthService) Register(username, password string) error {
	if len(username) < 3 || len(password) < 6 {
		return errors.New("username must be at least 3 characters and password at least 6 characters")
	}

	return s.userRepo.CreateUser(username, password)
}

func (s *AuthService) Login(username, password string) (string, error) {
	if !s.userRepo.ValidatePassword(username, password) {
		return "", errors.New("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok := claims["username"].(string)
		if !ok {
			return "", errors.New("invalid token claims")
		}
		return username, nil
	}

	return "", errors.New("invalid token")
}
