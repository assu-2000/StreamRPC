package config

import "os"

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

func LoadPostgresConfig() PostgresConfig {
	return PostgresConfig{
		Host:     getEnv("PG_HOST", "localhost"),
		Port:     getEnv("PG_PORT", "5432"),
		User:     getEnv("PG_USER", "postgres"),
		Password: getEnv("PG_PASSWORD", "postgres"),
		DBName:   getEnv("PG_DBNAME", "chatdb"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
