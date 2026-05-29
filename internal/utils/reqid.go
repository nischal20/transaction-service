package utils

import (
	"context"
	"log"
)

type requestIDKey struct{}

// NewRequestID returns a copy of ctx with the request ID stored inside.
func NewRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestIDFromCtx retrieves the request ID stored by NewRequestID.
// Returns "" if no ID is present.
func RequestIDFromCtx(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// Logf is a drop-in for log.Printf that prepends [<request-id>] when one is
// present in ctx. Use it at every layer so all log lines for a single request
// share the same ID and can be grepped together.
func Logf(ctx context.Context, format string, args ...any) {
	if id := RequestIDFromCtx(ctx); id != "" {
		log.Printf("["+id+"] "+format, args...)
	} else {
		log.Printf(format, args...)
	}
}
