package audit_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nischalpatel/transactions-api/internal/audit"
)

var insertAuditSQL = regexp.QuoteMeta(
	`INSERT INTO audit_logs (event_type, resource, resource_id, actor, request_id)
	 VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''))`,
)

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func beginTx(t *testing.T, db *sql.DB, mock sqlmock.Sqlmock) *sql.Tx {
	t.Helper()
	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)
	return tx
}

func TestLog_AccountCreated(t *testing.T) {
	db, mock := newMockDB(t)
	logger := audit.NewPostgresLogger(db)

	tx := beginTx(t, db, mock)
	mock.ExpectExec(insertAuditSQL).
		WithArgs("account.created", "account", int64(1), "user-42", "req-abc").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := logger.Log(context.Background(), tx, audit.Entry{
		EventType:  audit.EventAccountCreated,
		Resource:   "account",
		ResourceID: 1,
		Actor:      "user-42",
		RequestID:  "req-abc",
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLog_TransactionCreated(t *testing.T) {
	db, mock := newMockDB(t)
	logger := audit.NewPostgresLogger(db)

	tx := beginTx(t, db, mock)
	mock.ExpectExec(insertAuditSQL).
		WithArgs("transaction.created", "transaction", int64(99), "user-7", "req-xyz").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := logger.Log(context.Background(), tx, audit.Entry{
		EventType:  audit.EventTransactionCreated,
		Resource:   "transaction",
		ResourceID: 99,
		Actor:      "user-7",
		RequestID:  "req-xyz",
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLog_EmptyActorAndRequestID(t *testing.T) {
	db, mock := newMockDB(t)
	logger := audit.NewPostgresLogger(db)

	tx := beginTx(t, db, mock)
	mock.ExpectExec(insertAuditSQL).
		WithArgs("account.created", "account", int64(5), "", "").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := logger.Log(context.Background(), tx, audit.Entry{
		EventType:  audit.EventAccountCreated,
		Resource:   "account",
		ResourceID: 5,
		Actor:      "",
		RequestID:  "",
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLog_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	logger := audit.NewPostgresLogger(db)

	tx := beginTx(t, db, mock)
	mock.ExpectExec(insertAuditSQL).
		WithArgs("account.created", "account", int64(1), "user-1", "req-1").
		WillReturnError(errors.New("connection reset"))

	err := logger.Log(context.Background(), tx, audit.Entry{
		EventType:  audit.EventAccountCreated,
		Resource:   "account",
		ResourceID: 1,
		Actor:      "user-1",
		RequestID:  "req-1",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "audit log")
	require.NoError(t, mock.ExpectationsWereMet())
}
