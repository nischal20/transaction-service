package transaction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nischalpatel/transactions-api/internal/audit"
	"github.com/nischalpatel/transactions-api/internal/model"
	memaccount "github.com/nischalpatel/transactions-api/internal/repository/memory/account"
	memtransaction "github.com/nischalpatel/transactions-api/internal/repository/memory/transaction"
	svcaccount "github.com/nischalpatel/transactions-api/internal/service/account"
	svctransaction "github.com/nischalpatel/transactions-api/internal/service/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTxRepo is a TransactionRepository whose Create always returns a
// configurable error, letting us exercise the repo-error path in the service.
type stubTxRepo struct {
	createErr error
}

func (s *stubTxRepo) Create(ctx context.Context, accountID, operationTypeID int64, amount float64, txType string) (*model.Transaction, error) {
	return nil, s.createErr
}

func (s *stubTxRepo) FindOperationType(ctx context.Context, operationTypeID int64) (*model.OperationType, error) {
	return &model.OperationType{OperationTypeID: operationTypeID, Description: "stub"}, nil
}

func setupTxService(t *testing.T) (*svctransaction.TransactionService, int64) {
	t.Helper()
	accStore := memaccount.NewAccountStore()
	txStore := memtransaction.NewTransactionStore()

	accSvc := svcaccount.NewAccountService(accStore, audit.NoopLogger{})
	acc, err := accSvc.CreateAccount(context.Background(), "12345678900")
	require.NoError(t, err)

	txSvc := svctransaction.NewTransactionService(txStore, accStore, audit.NoopLogger{})
	return txSvc, acc.AccountID
}

// ── Amount sign per spec: debit = negative, credit = positive ─────────────────

func TestCreateTransaction_Purchase_StoredAsNegative(t *testing.T) {
	svc, accID := setupTxService(t)

	tx, err := svc.CreateTransaction(context.Background(), accID, model.OperationNormalPurchase, 100.0)
	require.NoError(t, err)
	assert.Equal(t, -100.0, tx.Amount)
}

func TestCreateTransaction_Withdrawal_StoredAsNegative(t *testing.T) {
	svc, accID := setupTxService(t)

	tx, err := svc.CreateTransaction(context.Background(), accID, model.OperationWithdrawal, 50.0)
	require.NoError(t, err)
	assert.Equal(t, -50.0, tx.Amount)
}

func TestCreateTransaction_PurchaseInstallments_StoredAsNegative(t *testing.T) {
	svc, accID := setupTxService(t)

	tx, err := svc.CreateTransaction(context.Background(), accID, model.OperationPurchaseInstallments, 75.0)
	require.NoError(t, err)
	assert.Equal(t, -75.0, tx.Amount)
}

func TestCreateTransaction_CreditVoucher_StoredAsPositive(t *testing.T) {
	svc, accID := setupTxService(t)

	tx, err := svc.CreateTransaction(context.Background(), accID, model.OperationCreditVoucher, 200.0)
	require.NoError(t, err)
	assert.Equal(t, 200.0, tx.Amount)
}

// ── Type field ────────────────────────────────────────────────────────────────

func TestCreateTransaction_TypeIsDebit_ForPurchase(t *testing.T) {
	svc, accID := setupTxService(t)

	tx, err := svc.CreateTransaction(context.Background(), accID, model.OperationNormalPurchase, 50.0)
	require.NoError(t, err)
	assert.Equal(t, "debit", tx.Type)
}

func TestCreateTransaction_TypeIsDebit_ForWithdrawal(t *testing.T) {
	svc, accID := setupTxService(t)

	tx, err := svc.CreateTransaction(context.Background(), accID, model.OperationWithdrawal, 30.0)
	require.NoError(t, err)
	assert.Equal(t, "debit", tx.Type)
}

func TestCreateTransaction_TypeIsCredit_ForCreditVoucher(t *testing.T) {
	svc, accID := setupTxService(t)

	tx, err := svc.CreateTransaction(context.Background(), accID, model.OperationCreditVoucher, 100.0)
	require.NoError(t, err)
	assert.Equal(t, "credit", tx.Type)
}

// ── Validation ────────────────────────────────────────────────────────────────

func TestCreateTransaction_ZeroAmount_Rejected(t *testing.T) {
	svc, accID := setupTxService(t)

	_, err := svc.CreateTransaction(context.Background(), accID, model.OperationNormalPurchase, 0)
	assert.EqualError(t, err, "amount must be greater than zero")
}

func TestCreateTransaction_NegativeAmount_Rejected(t *testing.T) {
	svc, accID := setupTxService(t)

	_, err := svc.CreateTransaction(context.Background(), accID, model.OperationNormalPurchase, -10.0)
	assert.EqualError(t, err, "amount must be greater than zero")
}

func TestCreateTransaction_AccountNotFound(t *testing.T) {
	svc, _ := setupTxService(t)

	_, err := svc.CreateTransaction(context.Background(), 999, model.OperationNormalPurchase, 10.0)
	assert.EqualError(t, err, "account not found")
}

func TestCreateTransaction_InvalidOperationType(t *testing.T) {
	svc, accID := setupTxService(t)

	_, err := svc.CreateTransaction(context.Background(), accID, 99, 10.0)
	assert.EqualError(t, err, "operation type not found")
}

func TestCreateTransaction_InvalidAccountID_Zero(t *testing.T) {
	svc, _ := setupTxService(t)

	_, err := svc.CreateTransaction(context.Background(), 0, model.OperationNormalPurchase, 10.0)
	assert.EqualError(t, err, "invalid account_id")
}

func TestCreateTransaction_InvalidAccountID_Negative(t *testing.T) {
	svc, _ := setupTxService(t)

	_, err := svc.CreateTransaction(context.Background(), -5, model.OperationNormalPurchase, 10.0)
	assert.EqualError(t, err, "invalid account_id")
}

// ── Misc ──────────────────────────────────────────────────────────────────────

func TestCreateTransaction_AllDebitTypesCovered(t *testing.T) {
	svc, accID := setupTxService(t)

	for _, opType := range []int64{
		model.OperationNormalPurchase,
		model.OperationPurchaseInstallments,
		model.OperationWithdrawal,
	} {
		tx, err := svc.CreateTransaction(context.Background(), accID, opType, 10.0)
		require.NoError(t, err, "op_type=%d", opType)
		assert.Equal(t, -10.0, tx.Amount, "op_type=%d amount must be negative", opType)
		assert.Equal(t, "debit", tx.Type, "op_type=%d must have type debit", opType)
	}
}

func TestCreateTransaction_TransactionIDsAreUnique(t *testing.T) {
	svc, accID := setupTxService(t)

	tx1, err := svc.CreateTransaction(context.Background(), accID, model.OperationNormalPurchase, 10.0)
	require.NoError(t, err)
	tx2, err := svc.CreateTransaction(context.Background(), accID, model.OperationCreditVoucher, 20.0)
	require.NoError(t, err)

	assert.NotEqual(t, tx1.TransactionID, tx2.TransactionID)
}

func TestCreateTransaction_EventDateIsSet(t *testing.T) {
	svc, accID := setupTxService(t)

	tx, err := svc.CreateTransaction(context.Background(), accID, model.OperationCreditVoucher, 50.0)
	require.NoError(t, err)
	assert.False(t, tx.EventDate.IsZero())
}

func TestCreateTransaction_RepoCreateError(t *testing.T) {
	accStore := memaccount.NewAccountStore()
	acc, err := accStore.Create(context.Background(), "12345678900")
	require.NoError(t, err)

	svc := svctransaction.NewTransactionService(&stubTxRepo{createErr: errors.New("storage failure")}, accStore, audit.NoopLogger{})

	_, err = svc.CreateTransaction(context.Background(), acc.AccountID, model.OperationNormalPurchase, 10.0)
	assert.EqualError(t, err, "storage failure")
}
