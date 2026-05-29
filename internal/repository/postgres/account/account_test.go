package account_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/nischalpatel/transactions-api/internal/repository"
	pgaccount "github.com/nischalpatel/transactions-api/internal/repository/postgres/account"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func TestCreate_RunsInsertReturningQuery(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(`INSERT INTO accounts \(document_number\) VALUES \(\$1\) RETURNING account_id, document_number`).
		WithArgs("12345678900").
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "document_number"}).
			AddRow(1, "12345678900"))

	store := pgaccount.NewAccountStore(db)
	acc, err := store.Create(context.Background(), "12345678900")

	require.NoError(t, err)
	assert.Equal(t, int64(1), acc.AccountID)
	assert.Equal(t, "12345678900", acc.DocumentNumber)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DuplicateDocument_ReturnsFriendlyError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(`INSERT INTO accounts`).
		WithArgs("12345678900").
		WillReturnError(&pq.Error{Code: "23505", Message: "duplicate key value violates unique constraint"})

	store := pgaccount.NewAccountStore(db)
	_, err := store.Create(context.Background(), "12345678900")

	assert.EqualError(t, err, "document_number already exists")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DBError_WrapsError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(`INSERT INTO accounts`).
		WithArgs("12345678900").
		WillReturnError(sql.ErrConnDone)

	store := pgaccount.NewAccountStore(db)
	_, err := store.Create(context.Background(), "12345678900")

	assert.ErrorContains(t, err, "insert account")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_RunsSelectQuery(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(`SELECT account_id, document_number FROM accounts WHERE account_id = \$1`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "document_number"}).
			AddRow(1, "12345678900"))

	store := pgaccount.NewAccountStore(db)
	acc, err := store.FindByID(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, int64(1), acc.AccountID)
	assert.Equal(t, "12345678900", acc.DocumentNumber)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_NotFound_ReturnsSentinelError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(`SELECT account_id, document_number FROM accounts WHERE account_id = \$1`).
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{"account_id", "document_number"}))

	store := pgaccount.NewAccountStore(db)
	_, err := store.FindByID(context.Background(), 999)

	assert.True(t, errors.Is(err, repository.ErrNotFound))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_DBError_WrapsError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(`SELECT account_id, document_number FROM accounts WHERE account_id = \$1`).
		WithArgs(int64(1)).
		WillReturnError(sql.ErrConnDone)

	store := pgaccount.NewAccountStore(db)
	_, err := store.FindByID(context.Background(), 1)

	assert.ErrorContains(t, err, "query account")
	assert.NoError(t, mock.ExpectationsWereMet())
}
