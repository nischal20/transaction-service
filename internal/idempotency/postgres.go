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
		`SELECT key, request_hash, transaction_id, created_at
		 FROM idempotency_keys WHERE key = $1`,
		key,
	).Scan(&r.Key, &r.RequestHash, &r.TransactionID, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find idempotency key: %w", err)
	}
	return &r, nil
}

func (s *PostgresStore) Save(ctx context.Context, tx *sql.Tx, key, requestHash string, transactionID int64) error {
	res, err := tx.ExecContext(ctx,
		`INSERT INTO idempotency_keys (key, request_hash, transaction_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (key) DO NOTHING`,
		key, requestHash, transactionID,
	)
	if err != nil {
		return fmt.Errorf("save idempotency key: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("save idempotency key: rows affected: %w", err)
	}
	if n == 0 {
		return ErrKeyConflict
	}
	return nil
}
