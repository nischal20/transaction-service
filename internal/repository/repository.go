package repository

import (
	"context"
	"errors"

	"github.com/nischalpatel/transactions-api/internal/model"
)

// ErrDuplicateDocument is returned by AccountRepository.Create when the
// document number is already registered.
var ErrDuplicateDocument = errors.New("document_number already exists")

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// AccountRepository defines persistence operations for accounts.
// Implementations must be safe for concurrent use.
type AccountRepository interface {
	// Create persists a new account with the given document number.
	// Returns ErrDuplicateDocument if the document number is already registered.
	Create(ctx context.Context, documentNumber string) (*model.Account, error)

	// FindByID retrieves an account by its numeric ID.
	// Returns ErrNotFound if no account with that ID exists.
	FindByID(ctx context.Context, accountID int64) (*model.Account, error)
}

// TransactionRepository defines persistence operations for transactions.
// Implementations must be safe for concurrent use.
type TransactionRepository interface {
	// Create records a new transaction against an account.
	// amount and txType are pre-computed by the service layer.
	Create(ctx context.Context, accountID, operationTypeID int64, amount float64, txType string) (*model.Transaction, error)

	// FindOperationType retrieves an operation type by its numeric ID.
	// Returns ErrNotFound if the operation type does not exist.
	FindOperationType(ctx context.Context, operationTypeID int64) (*model.OperationType, error)
}
