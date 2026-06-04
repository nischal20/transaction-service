package transaction

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/nischalpatel/transactions-api/internal/model"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// TransactionStore is a PostgreSQL-backed implementation of TransactionRepository.
type TransactionStore struct {
	db *sql.DB
}

func NewTransactionStore(db *sql.DB) *TransactionStore {
	return &TransactionStore{db: db}
}

func (s *TransactionStore) Create(ctx context.Context, tx *sql.Tx, accountID, operationTypeID int64, amount int64, txType string) (*model.Transaction, error) {
	utils.Logf(ctx, "repo[postgres]: insert transaction account_id=%d op_type=%d amount=%d", accountID, operationTypeID, amount)
	var t model.Transaction
	err := tx.QueryRowContext(ctx,
		`INSERT INTO transactions (account_id, operation_type_id, amount, type)
		 VALUES ($1, $2, $3, $4)
		 RETURNING transaction_id, account_id, operation_type_id, amount, type, event_date, created_at`,
		accountID, operationTypeID, amount, txType,
	).Scan(&t.TransactionID, &t.AccountID, &t.OperationTypeID, &t.Amount, &t.Type, &t.EventDate, &t.CreatedAt)
	if err != nil {
		utils.Logf(ctx, "repo[postgres]: insert transaction error: %v", err)
		return nil, fmt.Errorf("insert transaction: %w", err)
	}
	utils.Logf(ctx, "repo[postgres]: insert transaction ok transaction_id=%d type=%s amount=%d", t.TransactionID, t.Type, t.Amount)
	return &t, nil
}

func (s *TransactionStore) FindByID(ctx context.Context, transactionID int64) (*model.Transaction, error) {
	utils.Logf(ctx, "repo[postgres]: find transaction transaction_id=%d", transactionID)
	var t model.Transaction
	err := s.db.QueryRowContext(ctx,
		`SELECT transaction_id, account_id, operation_type_id, amount, type, event_date, created_at
		 FROM transactions WHERE transaction_id = $1`,
		transactionID,
	).Scan(&t.TransactionID, &t.AccountID, &t.OperationTypeID, &t.Amount, &t.Type, &t.EventDate, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("transaction %d not found", transactionID)
	}
	if err != nil {
		return nil, fmt.Errorf("find transaction: %w", err)
	}
	return &t, nil
}

func (s *TransactionStore) FindOperationType(ctx context.Context, operationTypeID int64) (*model.OperationType, error) {
	utils.Logf(ctx, "repo[postgres]: find operation_type op_type=%d", operationTypeID)
	var ot model.OperationType
	err := s.db.QueryRowContext(ctx,
		`SELECT operation_type_id, description FROM operation_types WHERE operation_type_id = $1`,
		operationTypeID,
	).Scan(&ot.OperationTypeID, &ot.Description)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Logf(ctx, "repo[postgres]: find operation_type not found op_type=%d", operationTypeID)
		return nil, errors.New("operation type not found")
	}
	if err != nil {
		utils.Logf(ctx, "repo[postgres]: find operation_type error: %v", err)
		return nil, fmt.Errorf("query operation type: %w", err)
	}
	utils.Logf(ctx, "repo[postgres]: find operation_type ok description=%q", ot.Description)
	return &ot, nil
}
