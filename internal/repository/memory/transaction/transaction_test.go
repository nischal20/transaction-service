package transaction_test

import (
	"context"
	"testing"

	"github.com/nischalpatel/transactions-api/internal/model"
	memtransaction "github.com/nischalpatel/transactions-api/internal/repository/memory/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionStore_Create_Success(t *testing.T) {
	store := memtransaction.NewTransactionStore()

	tx, err := store.Create(context.Background(), 1, model.OperationNormalPurchase, -100.0, "debit")
	require.NoError(t, err)
	assert.Greater(t, tx.TransactionID, int64(0))
	assert.Equal(t, int64(1), tx.AccountID)
	assert.Equal(t, int64(model.OperationNormalPurchase), tx.OperationTypeID)
	assert.Equal(t, -100.0, tx.Amount)
	assert.Equal(t, "debit", tx.Type)
	assert.False(t, tx.EventDate.IsZero())
}

func TestTransactionStore_Create_SetsTypeFromOperationType(t *testing.T) {
	store := memtransaction.NewTransactionStore()

	cases := []struct {
		opType       int64
		amount       float64
		txType       string
	}{
		{model.OperationNormalPurchase, -50.0, "debit"},
		{model.OperationPurchaseInstallments, -75.5, "debit"},
		{model.OperationWithdrawal, -200.0, "debit"},
		{model.OperationCreditVoucher, 300.0, "credit"},
	}

	for _, tc := range cases {
		tx, err := store.Create(context.Background(), 1, tc.opType, tc.amount, tc.txType)
		require.NoError(t, err)
		assert.Equal(t, tc.amount, tx.Amount, "op_type=%d", tc.opType)
		assert.Equal(t, tc.txType, tx.Type, "op_type=%d", tc.opType)
	}
}

func TestTransactionStore_Create_UniqueIDs(t *testing.T) {
	store := memtransaction.NewTransactionStore()

	tx1, err := store.Create(context.Background(), 1, model.OperationNormalPurchase, -10.0, "debit")
	require.NoError(t, err)
	tx2, err := store.Create(context.Background(), 1, model.OperationCreditVoucher, 20.0, "credit")
	require.NoError(t, err)

	assert.NotEqual(t, tx1.TransactionID, tx2.TransactionID)
}

func TestTransactionStore_FindOperationType_AllFour(t *testing.T) {
	store := memtransaction.NewTransactionStore()

	cases := []struct {
		id          int64
		description string
	}{
		{model.OperationNormalPurchase, "Normal Purchase"},
		{model.OperationPurchaseInstallments, "Purchase with installments"},
		{model.OperationWithdrawal, "Withdrawal"},
		{model.OperationCreditVoucher, "Credit Voucher"},
	}

	for _, tc := range cases {
		ot, err := store.FindOperationType(context.Background(), tc.id)
		require.NoError(t, err, "op_type=%d should exist", tc.id)
		assert.Equal(t, tc.id, ot.OperationTypeID)
		assert.Equal(t, tc.description, ot.Description)
	}
}

func TestTransactionStore_FindOperationType_Unknown(t *testing.T) {
	store := memtransaction.NewTransactionStore()

	_, err := store.FindOperationType(context.Background(), 99)
	assert.EqualError(t, err, "operation type not found")
}

func TestTransactionStore_FindOperationType_Zero(t *testing.T) {
	store := memtransaction.NewTransactionStore()

	_, err := store.FindOperationType(context.Background(), 0)
	assert.Error(t, err)
}
