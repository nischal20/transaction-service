package idempotency

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// PostgresStore is a PostgreSQL-backed idempotency repository.
type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Find(ctx context.Context, key string) (*Record, error) {
	var r Record
	err := s.db.QueryRowContext(ctx,
		`SELECT key, request_hash, response_code, response_body, created_at
		 FROM idempotency_keys WHERE key = $1`,
		key,
	).Scan(&r.Key, &r.RequestHash, &r.ResponseCode, &r.ResponseBody, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find idempotency key: %w", err)
	}
	return &r, nil
}

func (s *PostgresStore) Save(ctx context.Context, key, requestHash string, responseCode int, responseBody []byte) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO idempotency_keys (key, request_hash, response_code, response_body)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (key) DO NOTHING`,
		key, requestHash, responseCode, responseBody,
	)
	if err != nil {
		return fmt.Errorf("save idempotency key: %w", err)
	}
	return nil
}
