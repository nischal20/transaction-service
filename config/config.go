package config

import (
	"errors"
	"os"
)

// DBDriver defines which storage backend to use.
type DBDriver string

const (
	DBDriverMemory   DBDriver = "memory"
	DBDriverPostgres DBDriver = "postgres"
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
	DBDriver    DBDriver
	Postgres    PostgresConfig // only when DBDriver == postgres
	SwaggerPort string         // port for the Swagger UI server (default: 9001, set "" to disable)
}

// Load reads configuration from environment variables.
// Returns an error if DB_DRIVER=postgres but DB_HOST is not set.
func Load() (*Config, error) {
	driver := DBDriver(getEnv("DB_DRIVER", string(DBDriverMemory)))
	if driver != DBDriverMemory && driver != DBDriverPostgres {
		driver = DBDriverMemory
	}

	var pg PostgresConfig
	if driver == DBDriverPostgres {
		pg.Host = os.Getenv("DB_HOST")
		if pg.Host == "" {
			return nil, errors.New("DB_HOST must be set when DB_DRIVER=postgres")
		}
		pg.Port = os.Getenv("DB_PORT")
		if pg.Port == "" {
			return nil, errors.New("DB_PORT must be set when DB_DRIVER=postgres")
		}
		pg.User = os.Getenv("DB_USER")
		if pg.User == "" {
			return nil, errors.New("DB_USER must be set when DB_DRIVER=postgres")
		}
		pg.Password = os.Getenv("DB_PASSWORD")
		if pg.Password == "" {
			return nil, errors.New("DB_PASSWORD must be set when DB_DRIVER=postgres")
		}
		pg.DBName = os.Getenv("DB_NAME")
		if pg.DBName == "" {
			return nil, errors.New("DB_NAME must be set when DB_DRIVER=postgres")
		}
		pg.SSLMode = os.Getenv("DB_SSLMODE")
		if pg.SSLMode == "" {
			return nil, errors.New("DB_SSLMODE must be set when DB_DRIVER=postgres")
		}
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		DBDriver:    driver,
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
