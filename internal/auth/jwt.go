package auth

import (
	"errors"
	"github.com/assu-2000/StreamRPC/config"
	"github.com/google/uuid"

	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTService struct {
	secretKey       string
	accessDuration  time.Duration
	refreshDuration time.Duration
}

func NewJWTService(cfg config.JWTConfig) *JWTService {
	return &JWTService{
		secretKey:       cfg.SecretKey,
		accessDuration:  cfg.AccessDuration,
		refreshDuration: cfg.RefreshDuration,
	}
}

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

func (s *JWTService) GenerateTokens(userID uuid.UUID) (accessToken, refreshToken string, err error) {
	// Access token
	accessClaims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessDuration)),
		},
	}
	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.secretKey))
	if err != nil {
		return "", "", err
	}

	// Refresh token
	refreshClaims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshDuration)),
		},
	}

	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(s.secretKey))
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	ErrInvalidToken := errors.New("invalid token")
	return nil, ErrInvalidToken
}
