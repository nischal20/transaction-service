package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/audit"
	"github.com/nischalpatel/transactions-api/internal/idempotency"
	"github.com/nischalpatel/transactions-api/internal/model"
	accountrepo "github.com/nischalpatel/transactions-api/internal/repository/account"
	txrepo "github.com/nischalpatel/transactions-api/internal/repository/transaction"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// TransactionServicer is the interface handlers depend on for transaction operations.
type TransactionServicer interface {
	CreateTransaction(ctx context.Context, accountID, operationTypeID int64, amount int64, idemKey, idemHash string) (*model.Transaction, error)
}

// TransactionService handles business logic for transactions.
type TransactionService struct {
	txRepo  txrepo.Repository
	accRepo accountrepo.Repository
	auditor audit.Logger
	idem    idempotency.Repository
	db      *sql.DB
}

func NewTransactionService(
	txRepo txrepo.Repository,
	accRepo accountrepo.Repository,
	auditor audit.Logger,
	idem idempotency.Repository,
	db *sql.DB,
) *TransactionService {
	return &TransactionService{txRepo: txRepo, accRepo: accRepo, auditor: auditor, idem: idem, db: db}
}

// CreateTransaction validates input and persists the transaction.
//
// Idempotency is handled here:
//   - Same key + same body → return the existing transaction (no insert)
//   - Same key + different body → apperr.Conflict (→ 409)
//   - New key → insert transaction, save idempotency record, and audit log all
//     in one db transaction — atomic, no crash window between them
func (s *TransactionService) CreateTransaction(ctx context.Context, accountID, operationTypeID int64, amount int64, idemKey, idemHash string) (*model.Transaction, error) {
	utils.Logf(ctx, "service: create transaction: account_id=%d op_type=%d amount=%d", accountID, operationTypeID, amount)

	if accountID <= 0 {
		return nil, apperr.Validation("invalid account_id")
	}
	if amount <= 0 {
		return nil, apperr.Validation("amount must be greater than zero")
	}

	// Idempotency check — before any writes.
	if rec, err := s.idem.Find(ctx, idemKey); err != nil {
		utils.Logf(ctx, "service: create transaction: idempotency lookup error: %v", err)
		return nil, fmt.Errorf("idempotency lookup: %w", err)
	} else if rec != nil {
		if rec.RequestHash != idemHash {
			return nil, apperr.Conflict("idempotency key reused with a different request body")
		}
		utils.Logf(ctx, "service: create transaction: idempotency hit key=%s transaction_id=%d", idemKey, rec.TransactionID)
		return s.txRepo.FindByID(ctx, rec.TransactionID)
	}

	// Read-only lookups outside the db tx — they don't write anything.
	if _, err := s.accRepo.FindByID(ctx, accountID); err != nil {
		return nil, apperr.NotFound("account not found")
	}
	if _, err := s.txRepo.FindOperationType(ctx, operationTypeID); err != nil {
		return nil, apperr.NotFound("operation type not found")
	}

	txType := model.TransactionType(operationTypeID)
	if model.IsDebit(operationTypeID) {
		amount = -amount
	}

	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin db tx: %w", err)
	}

	tx, err := s.txRepo.Create(ctx, sqlTx, accountID, operationTypeID, amount, txType)
	if err != nil {
		_ = sqlTx.Rollback()
		return nil, err
	}

	if err := s.idem.Save(ctx, sqlTx, idemKey, idemHash, tx.TransactionID); err != nil {
		_ = sqlTx.Rollback()
		if errors.Is(err, idempotency.ErrKeyConflict) {
			// A concurrent request won the race — fetch and return their record.
			utils.Logf(ctx, "service: create transaction: idempotency race lost, returning winner key=%s", idemKey)
			rec, findErr := s.idem.Find(ctx, idemKey)
			if findErr != nil {
				return nil, fmt.Errorf("idempotency re-lookup after conflict: %w", findErr)
			}
			if rec == nil {
				// Should not happen: we just lost a conflict, the row must exist.
				return nil, fmt.Errorf("idempotency conflict but no record found for key=%s", idemKey)
			}
			return s.txRepo.FindByID(ctx, rec.TransactionID)
		}
		return nil, err
	}

	if err := s.auditor.Log(ctx, sqlTx, audit.Entry{
		EventType:  audit.EventTransactionCreated,
		Resource:   "transaction",
		ResourceID: tx.TransactionID,
		Actor:      utils.ActorFromCtx(ctx),
		RequestID:  utils.RequestIDFromCtx(ctx),
	}); err != nil {
		_ = sqlTx.Rollback()
		return nil, err
	}

	if err := sqlTx.Commit(); err != nil {
		return nil, fmt.Errorf("commit db tx: %w", err)
	}
	utils.Logf(ctx, "service: create transaction: committed transaction_id=%d", tx.TransactionID)
	return tx, nil
}
