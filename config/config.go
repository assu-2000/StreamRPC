package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func LoadPostgresConfig() *PostgresConfig {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return &PostgresConfig{
		Host:     os.Getenv("PG_HOST"),
		Port:     os.Getenv("PG_PORT"),
		User:     os.Getenv("PG_USER"),
		Password: os.Getenv("PG_PASSWORD"),
		DBName:   os.Getenv("PG_DBNAME"),
	}
}
