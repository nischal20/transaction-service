package transaction

import (
	"context"

	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/model"
	"github.com/nischalpatel/transactions-api/internal/repository"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// TransactionServicer is the interface handlers depend on for transaction operations.
// Handlers never import the concrete TransactionService — they depend on this
// interface, which allows stubs to be injected in tests without a real store.
type TransactionServicer interface {
	// CreateTransaction validates input and records a financial operation.
	// amount must be positive — direction is derived from operationTypeID, not the sign.
	// Returns apperr.ErrValidation for bad input, apperr.ErrNotFound if the account
	// or operation type does not exist.
	CreateTransaction(ctx context.Context, accountID, operationTypeID int64, amount float64) (*model.Transaction, error)

}

// TransactionService handles business logic for transactions.
type TransactionService struct {
	txRepo  repository.TransactionRepository
	accRepo repository.AccountRepository
}

func NewTransactionService(
	txRepo repository.TransactionRepository,
	accRepo repository.AccountRepository,
) *TransactionService {
	return &TransactionService{txRepo: txRepo, accRepo: accRepo}
}

// CreateTransaction validates input and persists the transaction.
//
// Callers must send a positive amount. The direction is conveyed by the
// type field ("debit" / "credit") derived from the operation type —
// no sign manipulation is applied to the amount.
func (s *TransactionService) CreateTransaction(ctx context.Context, accountID, operationTypeID int64, amount float64) (*model.Transaction, error) {
	utils.Logf(ctx, "service: create transaction: account_id=%d op_type=%d amount=%.2f", accountID, operationTypeID, amount)

	if accountID <= 0 {
		utils.Logf(ctx, "service: create transaction: validation failed: invalid account_id=%d", accountID)
		return nil, apperr.Validation("invalid account_id")
	}
	if amount <= 0 {
		utils.Logf(ctx, "service: create transaction: validation failed: amount must be positive, got %.2f", amount)
		return nil, apperr.Validation("amount must be greater than zero")
	}

	if _, err := s.accRepo.FindByID(ctx, accountID); err != nil {
		utils.Logf(ctx, "service: create transaction: account not found account_id=%d", accountID)
		return nil, apperr.NotFound("account not found")
	}

	if _, err := s.txRepo.FindOperationType(ctx, operationTypeID); err != nil {
		utils.Logf(ctx, "service: create transaction: operation type not found op_type=%d", operationTypeID)
		return nil, apperr.NotFound("operation type not found")
	}

	// Derive type and apply sign in one place per spec:
	// debit operations store negative amounts, credit store positive.
	txType := model.TransactionType(operationTypeID)
	if model.IsDebit(operationTypeID) {
		amount = -amount
	}

	utils.Logf(ctx, "service: create transaction: op_type=%d type=%s amount=%.2f", operationTypeID, txType, amount)

	tx, err := s.txRepo.Create(ctx, accountID, operationTypeID, amount, txType)
	if err != nil {
		utils.Logf(ctx, "service: create transaction: repo error: %v", err)
		return nil, err
	}
	utils.Logf(ctx, "service: create transaction: created transaction_id=%d type=%s", tx.TransactionID, tx.Type)
	return tx, nil
}

