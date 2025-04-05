package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshToken struct {
	TokenHash string
	UserID    uuid.UUID
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

type PostgresTokenRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTokenRepository(db *pgxpool.Pool) *PostgresTokenRepository {
	return &PostgresTokenRepository{db: db}
}

func (r *PostgresTokenRepository) StoreRefreshToken(ctx context.Context, tokenHash string, userID uuid.UUID, expiresAt time.Time) error {
	query := `
		INSERT INTO refresh_tokens (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (token_hash) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, tokenHash, userID, expiresAt)
	return err
}

func (r *PostgresTokenRepository) FindRefreshToken(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	query := `
		SELECT token_hash, user_id, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var token RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.TokenHash,
		&token.UserID,
		&token.ExpiresAt,
		&token.Revoked,
		&token.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (r *PostgresTokenRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE token_hash = $1
	`

	_, err := r.db.Exec(ctx, query, tokenHash)
	return err
}

func (r *PostgresTokenRepository) RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked = TRUE
		WHERE user_id = $1 AND revoked = FALSE
	`

	_, err := r.db.Exec(ctx, query, userID)
	return err
}
