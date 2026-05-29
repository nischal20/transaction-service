package audit

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nischalpatel/transactions-api/internal/utils"
)

// PostgresLogger writes audit entries to the audit_logs table.
type PostgresLogger struct {
	db *sql.DB
}

func NewPostgresLogger(db *sql.DB) *PostgresLogger {
	return &PostgresLogger{db: db}
}

func (l *PostgresLogger) Log(ctx context.Context, entry Entry) error {
	utils.Logf(ctx, "audit: log event_type=%s resource=%s resource_id=%d", entry.EventType, entry.Resource, entry.ResourceID)
	_, err := l.db.ExecContext(ctx,
		`INSERT INTO audit_logs (event_type, resource, resource_id, actor, request_id)
		 VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''))`,
		string(entry.EventType),
		entry.Resource,
		entry.ResourceID,
		entry.Actor,
		entry.RequestID,
	)
	if err != nil {
		utils.Logf(ctx, "audit: log failed event_type=%s resource=%s resource_id=%d error=%v", entry.EventType, entry.Resource, entry.ResourceID, err)
		return fmt.Errorf("audit log: %w", err)
	}
	utils.Logf(ctx, "audit: log ok event_type=%s resource=%s resource_id=%d", entry.EventType, entry.Resource, entry.ResourceID)
	return nil
}
