package idempotency_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nischalpatel/transactions-api/internal/idempotency"
)

var (
	selectIdemSQL = regexp.QuoteMeta(
		`SELECT key, request_hash, transaction_id, created_at
		 FROM idempotency_keys WHERE key = $1`,
	)
	insertIdemSQL = regexp.QuoteMeta(
		`INSERT INTO idempotency_keys (key, request_hash, transaction_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (key) DO NOTHING`,
	)
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

// ---- Find ------------------------------------------------------------------

func TestFind_RecordExists(t *testing.T) {
	db, mock := newMockDB(t)
	store := idempotency.NewPostgresStore(db)

	now := time.Now()
	mock.ExpectQuery(selectIdemSQL).
		WithArgs("key-abc").
		WillReturnRows(sqlmock.NewRows([]string{"key", "request_hash", "transaction_id", "created_at"}).
			AddRow("key-abc", "hash-123", int64(42), now))

	rec, err := store.Find(context.Background(), "key-abc")

	require.NoError(t, err)
	require.NotNil(t, rec)
	assert.Equal(t, "key-abc", rec.Key)
	assert.Equal(t, "hash-123", rec.RequestHash)
	assert.Equal(t, int64(42), rec.TransactionID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFind_RecordNotFound(t *testing.T) {
	db, mock := newMockDB(t)
	store := idempotency.NewPostgresStore(db)

	mock.ExpectQuery(selectIdemSQL).
		WithArgs("key-missing").
		WillReturnError(sql.ErrNoRows)

	rec, err := store.Find(context.Background(), "key-missing")

	require.NoError(t, err)
	assert.Nil(t, rec)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFind_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	store := idempotency.NewPostgresStore(db)

	mock.ExpectQuery(selectIdemSQL).
		WithArgs("key-abc").
		WillReturnError(errors.New("connection reset"))

	rec, err := store.Find(context.Background(), "key-abc")

	require.Error(t, err)
	assert.Nil(t, rec)
	assert.Contains(t, err.Error(), "find idempotency key")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---- Save ------------------------------------------------------------------

func TestSave_Success(t *testing.T) {
	db, mock := newMockDB(t)
	store := idempotency.NewPostgresStore(db)

	tx := beginTx(t, db, mock)
	mock.ExpectExec(insertIdemSQL).
		WithArgs("key-abc", "hash-123", int64(42)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.Save(context.Background(), tx, "key-abc", "hash-123", 42)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSave_KeyConflict(t *testing.T) {
	db, mock := newMockDB(t)
	store := idempotency.NewPostgresStore(db)

	tx := beginTx(t, db, mock)
	// ON CONFLICT DO NOTHING → 0 rows affected
	mock.ExpectExec(insertIdemSQL).
		WithArgs("key-abc", "hash-123", int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := store.Save(context.Background(), tx, "key-abc", "hash-123", 42)

	require.Error(t, err)
	assert.True(t, errors.Is(err, idempotency.ErrKeyConflict))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSave_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	store := idempotency.NewPostgresStore(db)

	tx := beginTx(t, db, mock)
	mock.ExpectExec(insertIdemSQL).
		WithArgs("key-abc", "hash-123", int64(42)).
		WillReturnError(errors.New("db timeout"))

	err := store.Save(context.Background(), tx, "key-abc", "hash-123", 42)

	require.Error(t, err)
	assert.NotErrorIs(t, err, idempotency.ErrKeyConflict)
	assert.Contains(t, err.Error(), "save idempotency key")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSave_RowsAffectedError(t *testing.T) {
	db, mock := newMockDB(t)
	store := idempotency.NewPostgresStore(db)

	tx := beginTx(t, db, mock)
	mock.ExpectExec(insertIdemSQL).
		WithArgs("key-abc", "hash-123", int64(42)).
		WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected unavailable")))

	err := store.Save(context.Background(), tx, "key-abc", "hash-123", 42)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "rows affected")
	require.NoError(t, mock.ExpectationsWereMet())
}
