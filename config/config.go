package config

import (
	"errors"
	"os"
)

// PostgresConfig holds individual connection parameters for PostgreSQL.
type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// Config holds all application configuration sourced from environment variables.
type Config struct {
	Port        string
	Postgres    PostgresConfig
	SwaggerPort string // set "" to disable in production
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	pg := PostgresConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	if pg.Host == "" {
		return nil, errors.New("DB_HOST must be set")
	}
	if pg.Port == "" {
		return nil, errors.New("DB_PORT must be set")
	}
	if pg.User == "" {
		return nil, errors.New("DB_USER must be set")
	}
	if pg.Password == "" {
		return nil, errors.New("DB_PASSWORD must be set")
	}
	if pg.DBName == "" {
		return nil, errors.New("DB_NAME must be set")
	}
	if pg.SSLMode == "" {
		return nil, errors.New("DB_SSLMODE must be set")
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		Postgres:    pg,
		SwaggerPort: getEnv("SWAGGER_PORT", "9001"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
