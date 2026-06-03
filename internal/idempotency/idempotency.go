package idempotency

import (
	"context"
	"time"
)

// Record is a cached HTTP response stored for a given idempotency key.
type Record struct {
	Key          string
	RequestHash  string // SHA-256 of the raw request body
	ResponseCode int
	ResponseBody []byte
	CreatedAt    time.Time
}

// Repository stores and retrieves idempotency records.
type Repository interface {
	// Find returns the cached record for key, or (nil, nil) when not found.
	Find(ctx context.Context, key string) (*Record, error)
	// Save persists a new record. If the key already exists it is a no-op.
	Save(ctx context.Context, key, requestHash string, responseCode int, responseBody []byte) error
}
