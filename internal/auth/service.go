package auth

import (
	"context"
	"errors"
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserRepo interface {
	CreateUser(ctx context.Context, user *User) error
	FindUserByUsername(ctx context.Context, username string) (*User, error)
}
type AuthService struct {
	repo         UserRepo
	jwtService   *JWTService
	tokenService *TokenService
}

func NewAuthService(repo UserRepo, jwt *JWTService, token *TokenService) *AuthService {
	return &AuthService{repo: repo,
		jwtService:   jwt,
		tokenService: token,
	}
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
		ID:       uuid.New(),
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
	}

	return s.repo.CreateUser(context.Background(), user)
}

func (s *AuthService) Login(username, password string) (*User, string, string, error) {
	user, err := s.repo.FindUserByUsername(context.Background(), username)
	if err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, "", "", errors.New("invalid credentials")
	}

	//accessToken, refreshToken, err := s.jwtService.GenerateTokens(user.ID, username)
	//if err != nil {
	//	log.Printf("Failed to generate tokens: %v", err)
	//	return nil, "", "", errors.New("failed to generate tokens")
	//}
	accessToken, refreshToken, err := s.tokenService.GenerateAndStoreTokens(context.Background(), user.ID)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}

func (s *AuthService) HandleRefresh(ctx context.Context, refreshToken string) (string, string, error) {
	newAccessToken, newRefreshToken, err := s.tokenService.RefreshTokens(ctx, refreshToken)
	if err != nil {
		return "", "", err
	}
	return newAccessToken, newRefreshToken, nil
}

func (s *AuthService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	err := s.tokenService.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		return err
	}
	return nil
}

func (s *AuthService) RevokeAllTokens(ctx context.Context, userId uuid.UUID) error {
	err := s.tokenService.RevokeAllTokens(ctx, userId)
	if err != nil {
		return err
	}
	return nil
}
