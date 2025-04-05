package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TokenService struct {
	repo          TokenRepository
	jwtService    *JWTService
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

type TokenRepository interface {
	StoreRefreshToken(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time) error
	FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error
}

func NewTokenService(repo TokenRepository, jwtService *JWTService, accessExpiry, refreshExpiry time.Duration) *TokenService {
	return &TokenService{
		repo:          repo,
		jwtService:    jwtService,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

func (s *TokenService) GenerateAndStoreTokens(ctx context.Context, userID uuid.UUID) (accessToken, refreshToken string, err error) {
	// Génération des tokens
	accessToken, refreshToken, err = s.jwtService.GenerateTokens(userID)
	if err != nil {
		return "", "", err
	}

	refreshToken = uuid.New().String()
	refreshTokenHash := hashToken(refreshToken)

	// Stockage du refresh token
	expiresAt := time.Now().Add(s.refreshExpiry)
	if err := s.repo.StoreRefreshToken(ctx, refreshTokenHash, userID, expiresAt); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *TokenService) RefreshTokens(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, err error) {
	tokenHash := hashToken(refreshToken)

	storedToken, err := s.repo.FindRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", errors.New("invalid refresh token")
		}
		return "", "", err
	}

	if storedToken.Revoked || time.Now().After(storedToken.ExpiresAt) {
		return "", "", errors.New("invalid refresh token")
	}

	if err := s.repo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return "", "", err
	}

	return s.GenerateAndStoreTokens(ctx, storedToken.UserID)
}

func (s *TokenService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	return s.repo.RevokeRefreshToken(ctx, hashToken(refreshToken))
}

func (s *TokenService) RevokeAllTokens(ctx context.Context, userID uuid.UUID) error {
	return s.repo.RevokeAllRefreshTokens(ctx, userID)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
