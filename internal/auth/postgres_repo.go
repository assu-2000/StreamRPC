package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.Password,
	)

	if err != nil {
		if isDuplicateKeyError(err) {
			return errors.New("username or email already exists")
		}
		fmt.Println(err)
		return err
	}

	return nil
}

func (r *PostgresRepository) FindUserByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash
		FROM users
		WHERE username = $1
	`

	var user User
	err := r.db.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func isDuplicateKeyError(err error) bool {
	return err.Error() == "ERROR: duplicate key value violates unique constraint"
}

type User struct {
	ID       uuid.UUID
	Username string
	Password string
	Email    string
}
