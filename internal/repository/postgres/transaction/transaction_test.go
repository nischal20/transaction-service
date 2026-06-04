package transaction_test

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

	pgtransaction "github.com/nischalpatel/transactions-api/internal/repository/postgres/transaction"
)

var (
	insertTransactionSQL = regexp.QuoteMeta(
		`INSERT INTO transactions (account_id, operation_type_id, amount, type)
		 VALUES ($1, $2, $3, $4)
		 RETURNING transaction_id, account_id, operation_type_id, amount, type, event_date, created_at`,
	)
	selectTransactionSQL = regexp.QuoteMeta(
		`SELECT transaction_id, account_id, operation_type_id, amount, type, event_date, created_at
		 FROM transactions WHERE transaction_id = $1`,
	)
	selectOperationTypeSQL = regexp.QuoteMeta(
		`SELECT operation_type_id, description FROM operation_types WHERE operation_type_id = $1`,
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
	store := pgtransaction.NewTransactionStore(db)

	now := time.Now()
	tx := beginTx(t, db, mock)
	mock.ExpectQuery(insertTransactionSQL).
		WithArgs(int64(1), int64(1), int64(-10000), "debit").
		WillReturnRows(sqlmock.NewRows([]string{
			"transaction_id", "account_id", "operation_type_id", "amount", "type", "event_date", "created_at",
		}).AddRow(42, 1, 1, -10000, "debit", now, now))

	t1, err := store.Create(context.Background(), tx, 1, 1, -10000, "debit")

	require.NoError(t, err)
	assert.Equal(t, int64(42), t1.TransactionID)
	assert.Equal(t, int64(1), t1.AccountID)
	assert.Equal(t, int64(1), t1.OperationTypeID)
	assert.Equal(t, int64(-10000), t1.Amount)
	assert.Equal(t, "debit", t1.Type)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_CreditVoucher(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgtransaction.NewTransactionStore(db)

	now := time.Now()
	tx := beginTx(t, db, mock)
	mock.ExpectQuery(insertTransactionSQL).
		WithArgs(int64(2), int64(4), int64(5000), "credit").
		WillReturnRows(sqlmock.NewRows([]string{
			"transaction_id", "account_id", "operation_type_id", "amount", "type", "event_date", "created_at",
		}).AddRow(7, 2, 4, 5000, "credit", now, now))

	t1, err := store.Create(context.Background(), tx, 2, 4, 5000, "credit")

	require.NoError(t, err)
	assert.Equal(t, int64(7), t1.TransactionID)
	assert.Equal(t, "credit", t1.Type)
	assert.Equal(t, int64(5000), t1.Amount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreate_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgtransaction.NewTransactionStore(db)

	tx := beginTx(t, db, mock)
	mock.ExpectQuery(insertTransactionSQL).
		WithArgs(int64(1), int64(1), int64(-10000), "debit").
		WillReturnError(errors.New("connection reset"))

	_, err := store.Create(context.Background(), tx, 1, 1, -10000, "debit")

	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---- FindByID --------------------------------------------------------------

func TestFindByID_Success(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgtransaction.NewTransactionStore(db)

	now := time.Now()
	mock.ExpectQuery(selectTransactionSQL).
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"transaction_id", "account_id", "operation_type_id", "amount", "type", "event_date", "created_at",
		}).AddRow(42, 1, 1, -10000, "debit", now, now))

	t1, err := store.FindByID(context.Background(), 42)

	require.NoError(t, err)
	assert.Equal(t, int64(42), t1.TransactionID)
	assert.Equal(t, int64(1), t1.AccountID)
	assert.Equal(t, "debit", t1.Type)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgtransaction.NewTransactionStore(db)

	mock.ExpectQuery(selectTransactionSQL).
		WithArgs(int64(999)).
		WillReturnError(sql.ErrNoRows)

	_, err := store.FindByID(context.Background(), 999)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgtransaction.NewTransactionStore(db)

	mock.ExpectQuery(selectTransactionSQL).
		WithArgs(int64(1)).
		WillReturnError(errors.New("timeout"))

	_, err := store.FindByID(context.Background(), 1)

	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---- FindOperationType -----------------------------------------------------

func TestFindOperationType_Success(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgtransaction.NewTransactionStore(db)

	mock.ExpectQuery(selectOperationTypeSQL).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"operation_type_id", "description"}).
			AddRow(1, "Normal Purchase"))

	ot, err := store.FindOperationType(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, int64(1), ot.OperationTypeID)
	assert.Equal(t, "Normal Purchase", ot.Description)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOperationType_AllTypes(t *testing.T) {
	cases := []struct {
		id   int64
		desc string
	}{
		{1, "Normal Purchase"},
		{2, "Purchase with installments"},
		{3, "Withdrawal"},
		{4, "Credit Voucher"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			db, mock := newMockDB(t)
			store := pgtransaction.NewTransactionStore(db)

			mock.ExpectQuery(selectOperationTypeSQL).
				WithArgs(tc.id).
				WillReturnRows(sqlmock.NewRows([]string{"operation_type_id", "description"}).
					AddRow(tc.id, tc.desc))

			ot, err := store.FindOperationType(context.Background(), tc.id)

			require.NoError(t, err)
			assert.Equal(t, tc.id, ot.OperationTypeID)
			assert.Equal(t, tc.desc, ot.Description)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFindOperationType_NotFound(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgtransaction.NewTransactionStore(db)

	mock.ExpectQuery(selectOperationTypeSQL).
		WithArgs(int64(99)).
		WillReturnError(sql.ErrNoRows)

	_, err := store.FindOperationType(context.Background(), 99)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "operation type not found")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOperationType_DBError(t *testing.T) {
	db, mock := newMockDB(t)
	store := pgtransaction.NewTransactionStore(db)

	mock.ExpectQuery(selectOperationTypeSQL).
		WithArgs(int64(1)).
		WillReturnError(errors.New("timeout"))

	_, err := store.FindOperationType(context.Background(), 1)

	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
