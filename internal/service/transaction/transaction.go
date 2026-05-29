package transaction

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/audit"
	"github.com/nischalpatel/transactions-api/internal/model"
	accountrepo "github.com/nischalpatel/transactions-api/internal/repository/account"
	txrepo "github.com/nischalpatel/transactions-api/internal/repository/transaction"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// TransactionServicer is the interface handlers depend on for transaction operations.
type TransactionServicer interface {
	CreateTransaction(ctx context.Context, accountID, operationTypeID int64, amount float64) (*model.Transaction, error)
}

// TransactionService handles business logic for transactions.
type TransactionService struct {
	txRepo  txrepo.Repository
	accRepo accountrepo.Repository
	auditor audit.Logger
	db      *sql.DB // non-nil in postgres mode; used to begin db transactions
}

func NewTransactionService(
	txRepo txrepo.Repository,
	accRepo accountrepo.Repository,
	auditor audit.Logger,
	db *sql.DB,
) *TransactionService {
	return &TransactionService{txRepo: txRepo, accRepo: accRepo, auditor: auditor, db: db}
}

// CreateTransaction validates input and persists the transaction.
//
// In PostgreSQL mode the transaction row and its audit entry are written inside
// a single database transaction — both commit or both roll back, so the audit
// trail is always consistent with the transactions table.
//
// Callers must send a positive amount. The direction is conveyed by the
// type field ("debit" / "credit") derived from the operation type.
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

	// Read-only lookups — outside the db tx since they don't write anything.
	if _, err := s.accRepo.FindByID(ctx, accountID); err != nil {
		utils.Logf(ctx, "service: create transaction: account not found account_id=%d", accountID)
		return nil, apperr.NotFound("account not found")
	}
	if _, err := s.txRepo.FindOperationType(ctx, operationTypeID); err != nil {
		utils.Logf(ctx, "service: create transaction: operation type not found op_type=%d", operationTypeID)
		return nil, apperr.NotFound("operation type not found")
	}

	txType := model.TransactionType(operationTypeID)
	if model.IsDebit(operationTypeID) {
		amount = -amount
	}
	utils.Logf(ctx, "service: create transaction: op_type=%d type=%s amount=%.2f", operationTypeID, txType, amount)

	if s.db != nil {
		return s.createWithDBTx(ctx, accountID, operationTypeID, amount, txType)
	}
	return s.createInMemory(ctx, accountID, operationTypeID, amount, txType)
}

func (s *TransactionService) createWithDBTx(ctx context.Context, accountID, operationTypeID int64, amount float64, txType string) (*model.Transaction, error) {
	utils.Logf(ctx, "service: create transaction: begin db tx")
	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin db tx: %w", err)
	}

	tx, err := s.txRepo.Create(ctx, sqlTx, accountID, operationTypeID, amount, txType)
	if err != nil {
		_ = sqlTx.Rollback()
		utils.Logf(ctx, "service: create transaction: rollback db tx: repo error: %v", err)
		return nil, err
	}
	utils.Logf(ctx, "service: create transaction: created transaction_id=%d type=%s", tx.TransactionID, tx.Type)

	if err := s.auditor.Log(ctx, sqlTx, audit.Entry{
		EventType:  audit.EventTransactionCreated,
		Resource:   "transaction",
		ResourceID: tx.TransactionID,
		RequestID:  utils.RequestIDFromCtx(ctx),
	}); err != nil {
		_ = sqlTx.Rollback()
		utils.Logf(ctx, "service: create transaction: rollback db tx: audit error: %v", err)
		return nil, err
	}

	if err := sqlTx.Commit(); err != nil {
		utils.Logf(ctx, "service: create transaction: commit failed: %v", err)
		return nil, fmt.Errorf("commit db tx: %w", err)
	}
	utils.Logf(ctx, "service: create transaction: committed db tx transaction_id=%d", tx.TransactionID)
	return tx, nil
}

func (s *TransactionService) createInMemory(ctx context.Context, accountID, operationTypeID int64, amount float64, txType string) (*model.Transaction, error) {
	tx, err := s.txRepo.Create(ctx, nil, accountID, operationTypeID, amount, txType)
	if err != nil {
		utils.Logf(ctx, "service: create transaction: repo error: %v", err)
		return nil, err
	}
	utils.Logf(ctx, "service: create transaction: created transaction_id=%d type=%s", tx.TransactionID, tx.Type)
	if err := s.auditor.Log(ctx, nil, audit.Entry{
		EventType:  audit.EventTransactionCreated,
		Resource:   "transaction",
		ResourceID: tx.TransactionID,
		RequestID:  utils.RequestIDFromCtx(ctx),
	}); err != nil {
		utils.Logf(ctx, "service: create transaction: audit log warning (non-fatal): %v", err)
	}
	return tx, nil
}
