package audit

import "context"

// NoopLogger is used in in-memory mode where there is no persistent DB.
type NoopLogger struct{}

func (NoopLogger) Log(_ context.Context, _ Entry) error { return nil }
