// Package audit provides an append-only audit trail for state-changing operations.
// Every account or transaction creation is recorded with its event type,
// affected resource, and the request ID for cross-log tracing.
// The actor field is nullable until authentication is added.
package audit

import "context"

// EventType identifies what happened.
type EventType string

const (
	EventAccountCreated     EventType = "account.created"
	EventTransactionCreated EventType = "transaction.created"
)

// Entry is a single audit record.
type Entry struct {
	EventType  EventType
	Resource   string // e.g. "account", "transaction"
	ResourceID int64
	Actor      string // empty until auth is wired in
	RequestID  string
}

// Logger writes audit entries. Implementations must be safe for concurrent use.
type Logger interface {
	// Log records an audit entry. Returns an error if the write fails.
	Log(ctx context.Context, entry Entry) error
}
