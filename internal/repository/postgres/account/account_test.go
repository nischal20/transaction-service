package account_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	accountrepo "github.com/nischalpatel/transactions-api/internal/repository/account"
	pgaccount "github.com/nischalpatel/transactions-api/internal/repository/postgres/account"
)

var (
	insertAccountSQL = regexp.QuoteMeta(
		`INSERT INTO accounts (document_number) VALUES ($1) RETURNING account_id, document_number, created_at, updated_at`,
	)
	selectAccountSQL = regexp.QuoteMeta(
		`SELECT account_id, document_number, created_at, updated_at FROM accounts WHERE account_id = $1`,
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

// ---- Create ----------------------------------------------------------------

func TestCreate_Success(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgaccount.NewAccountStore(db)

	now := time.Now()
	tx := beginTx(t, db, mock)
	mock.ExpectQuery(insertAccountSQL).
		WithArgs("12345678900").
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "document_number", "created_at", "updated_at"}).
			AddRow(1, "12345678900", now, now))

	acc, err := store.Create(context.Background(), tx, "12345678900")

	require.NoError(t, err)
	assert.Equal(t, int64(1), acc.AccountID)
	assert.Equal(t, "12345678900", acc.DocumentNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DuplicateDocument(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgaccount.NewAccountStore(db)

	tx := beginTx(t, db, mock)
	mock.ExpectQuery(insertAccountSQL).
		WithArgs("12345678900").
		WillReturnError(&pq.Error{Code: "23505"})

	_, err := store.Create(context.Background(), tx, "12345678900")

	require.Error(t, err)
	assert.True(t, errors.Is(err, accountrepo.ErrDuplicateDocument))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgaccount.NewAccountStore(db)

	tx := beginTx(t, db, mock)
	mock.ExpectQuery(insertAccountSQL).
		WithArgs("99999999999").
		WillReturnError(errors.New("connection reset"))

	_, err := store.Create(context.Background(), tx, "99999999999")

	require.Error(t, err)
	assert.NotErrorIs(t, err, accountrepo.ErrDuplicateDocument)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---- FindByID --------------------------------------------------------------

func TestFindByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgaccount.NewAccountStore(db)

	now := time.Now()
	mock.ExpectQuery(selectAccountSQL).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "document_number", "created_at", "updated_at"}).
			AddRow(1, "12345678900", now, now))

	acc, err := store.FindByID(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, int64(1), acc.AccountID)
	assert.Equal(t, "12345678900", acc.DocumentNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgaccount.NewAccountStore(db)

	mock.ExpectQuery(selectAccountSQL).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err := store.FindByID(context.Background(), 999)

	require.Error(t, err)
	assert.True(t, errors.Is(err, accountrepo.ErrNotFound))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgaccount.NewAccountStore(db)

	mock.ExpectQuery(selectAccountSQL).
		WithArgs(int64(1)).
		WillReturnError(errors.New("timeout"))

	_, err := store.FindByID(context.Background(), 1)

	require.Error(t, err)
	assert.NotErrorIs(t, err, accountrepo.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
