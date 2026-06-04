package idempotency

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ErrKeyConflict is returned by Save when the key was already inserted by a
// concurrent request (ON CONFLICT DO NOTHING → 0 rows affected).
var ErrKeyConflict = errors.New("idempotency key already exists")

// Record is the idempotency entry stored for a given key.
type Record struct {
	Key           string
	RequestHash   string // SHA-256 of the raw request body
	TransactionID int64
	CreatedAt     time.Time
}

// Repository stores and retrieves idempotency records.
type Repository interface {
	// Find returns the record for key, or (nil, nil) when not found.
	Find(ctx context.Context, key string) (*Record, error)
	// Save persists a new record inside tx — must be called within the same
	// database transaction as the business insert so both are atomic.
	Save(ctx context.Context, tx *sql.Tx, key, requestHash string, transactionID int64) error
}
