package auth

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo *PostgresRepository
}

func NewAuthService(repo *PostgresRepository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) Register(username, password, email string) error {
	// basic validation
	if username == "" || password == "" {
		return errors.New("username and password are required")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		return errors.New("failed to create user")
	}

	user := &User{
		ID:       uuid.New().String(),
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
	}

	return s.repo.CreateUser(context.Background(), user)
}

func (s *AuthService) Login(username, password string) (*User, error) {
	user, err := s.repo.FindUserByUsername(context.Background(), username)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	return user, nil
}
