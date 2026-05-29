package transaction_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/nischalpatel/transactions-api/internal/model"
	pgtransaction "github.com/nischalpatel/transactions-api/internal/repository/postgres/transaction"
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

func TestTransactionCreate_RunsInsertReturningQuery(t *testing.T) {
	db, mock := newMock(t)
	eventDate := time.Now().UTC()

	mock.ExpectQuery(`INSERT INTO transactions \(account_id, operation_type_id, amount, type\)\s+VALUES \(\$1, \$2, \$3, \$4\)\s+RETURNING transaction_id, account_id, operation_type_id, amount, type, event_date`).
		WithArgs(int64(1), int64(model.OperationNormalPurchase), -100.0, "debit").
		WillReturnRows(sqlmock.NewRows([]string{"transaction_id", "account_id", "operation_type_id", "amount", "type", "event_date"}).
			AddRow(1, 1, model.OperationNormalPurchase, -100.0, "debit", eventDate))

	store := pgtransaction.NewTransactionStore(db)
	tx, err := store.Create(context.Background(), 1, model.OperationNormalPurchase, -100.0, "debit")

	require.NoError(t, err)
	assert.Equal(t, int64(1), tx.TransactionID)
	assert.Equal(t, -100.0, tx.Amount)
	assert.Equal(t, "debit", tx.Type)
	assert.Equal(t, eventDate, tx.EventDate)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionCreate_CreditVoucher(t *testing.T) {
	db, mock := newMock(t)
	eventDate := time.Now().UTC()

	mock.ExpectQuery(`INSERT INTO transactions`).
		WithArgs(int64(2), int64(model.OperationCreditVoucher), 50.0, "credit").
		WillReturnRows(sqlmock.NewRows([]string{"transaction_id", "account_id", "operation_type_id", "amount", "type", "event_date"}).
			AddRow(2, 2, model.OperationCreditVoucher, 50.0, "credit", eventDate))

	store := pgtransaction.NewTransactionStore(db)
	tx, err := store.Create(context.Background(), 2, model.OperationCreditVoucher, 50.0, "credit")

	require.NoError(t, err)
	assert.Equal(t, 50.0, tx.Amount)
	assert.Equal(t, "credit", tx.Type)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTransactionCreate_DBError_WrapsError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(`INSERT INTO transactions`).
		WithArgs(int64(1), int64(model.OperationNormalPurchase), -100.0, "debit").
		WillReturnError(sql.ErrConnDone)

	store := pgtransaction.NewTransactionStore(db)
	_, err := store.Create(context.Background(), 1, model.OperationNormalPurchase, -100.0, "debit")

	assert.ErrorContains(t, err, "insert transaction")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOperationType_RunsSelectQuery(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(`SELECT operation_type_id, description FROM operation_types WHERE operation_type_id = \$1`).
		WithArgs(int64(model.OperationNormalPurchase)).
		WillReturnRows(sqlmock.NewRows([]string{"operation_type_id", "description"}).
			AddRow(model.OperationNormalPurchase, "Normal Purchase"))

	store := pgtransaction.NewTransactionStore(db)
	ot, err := store.FindOperationType(context.Background(), model.OperationNormalPurchase)

	require.NoError(t, err)
	assert.Equal(t, int64(model.OperationNormalPurchase), ot.OperationTypeID)
	assert.Equal(t, "Normal Purchase", ot.Description)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOperationType_AllFourTypes(t *testing.T) {
	cases := []struct {
		id   int64
		desc string
	}{
		{model.OperationNormalPurchase, "Normal Purchase"},
		{model.OperationPurchaseInstallments, "Purchase with installments"},
		{model.OperationWithdrawal, "Withdrawal"},
		{model.OperationCreditVoucher, "Credit Voucher"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			db, mock := newMock(t)

			mock.ExpectQuery(`SELECT operation_type_id, description FROM operation_types WHERE operation_type_id = \$1`).
				WithArgs(tc.id).
				WillReturnRows(sqlmock.NewRows([]string{"operation_type_id", "description"}).
					AddRow(tc.id, tc.desc))

			store := pgtransaction.NewTransactionStore(db)
			ot, err := store.FindOperationType(context.Background(), tc.id)

			require.NoError(t, err)
			assert.Equal(t, tc.id, ot.OperationTypeID)
			assert.Equal(t, tc.desc, ot.Description)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestFindOperationType_NotFound(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(`SELECT operation_type_id, description FROM operation_types WHERE operation_type_id = \$1`).
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"operation_type_id", "description"}))

	store := pgtransaction.NewTransactionStore(db)
	_, err := store.FindOperationType(context.Background(), 99)

	assert.EqualError(t, err, "operation type not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindOperationType_DBError_WrapsError(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectQuery(`SELECT operation_type_id, description FROM operation_types WHERE operation_type_id = \$1`).
		WithArgs(int64(1)).
		WillReturnError(sql.ErrConnDone)

	store := pgtransaction.NewTransactionStore(db)
	_, err := store.FindOperationType(context.Background(), 1)

	assert.ErrorContains(t, err, "query operation type")
	assert.NoError(t, mock.ExpectationsWereMet())
}
