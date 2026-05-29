package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresConfig holds the connection parameters for a PostgreSQL database.
type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// PostgresConnector implements Connector for PostgreSQL.
type PostgresConnector struct {
	cfg PostgresConfig
}

func NewPostgresConnector(cfg PostgresConfig) *PostgresConnector {
	return &PostgresConnector{cfg: cfg}
}

func (c *PostgresConnector) Connect() (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.cfg.Host, c.cfg.Port, c.cfg.User, c.cfg.Password, c.cfg.DBName, c.cfg.SSLMode,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres connection: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
