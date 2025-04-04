package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func NewPostgresConnection(cfg *PostgresConfig) (*pgxpool.Pool, error) {
	fmt.Println("Connecting to PostgreSQL")
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pg config: %w", err)
	}

	config.MaxConns = 10
	config.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pg pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	checkCurrentDatabase(pool)

	log.Println("Successfully connected to PostgreSQL")
	return pool, nil
}

func checkCurrentDatabase(pool *pgxpool.Pool) {
	var dbName string
	err := pool.QueryRow(context.Background(), "SELECT current_database()").Scan(&dbName)
	if err != nil {
		log.Printf("Error querying current database: %v", err)
		return
	}
	log.Printf("Connected to database: %s", dbName)
}
