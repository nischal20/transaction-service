package audit

import (
	"context"
	"database/sql"
)

// NoopLogger is used in in-memory mode where there is no persistent DB.
type NoopLogger struct{}

func (NoopLogger) Log(_ context.Context, _ *sql.Tx, _ Entry) error { return nil }
