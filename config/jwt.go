package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

type JWTConfig struct {
	SecretKey       string
	AccessDuration  time.Duration
	RefreshDuration time.Duration
}

func LoadJWTConfig() JWTConfig {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	return JWTConfig{
		SecretKey:       os.Getenv("JWT_SECRET_KEY"),
		AccessDuration:  15 * time.Minute,
		RefreshDuration: 7 * 24 * time.Hour,
	}
}
