package transaction

import (
	"context"
	"database/sql"

	"github.com/nischalpatel/transactions-api/internal/model"
)

// Repository defines persistence operations for transactions.
// Implementations must be safe for concurrent use.
type Repository interface {
	// Create records a new transaction against an account.
	// tx must be a live transaction when the insert should be part of a larger
	// atomic operation; pass nil to run outside any transaction.
	// amount and txType are pre-computed by the service layer.
	Create(ctx context.Context, tx *sql.Tx, accountID, operationTypeID int64, amount float64, txType string) (*model.Transaction, error)

	// FindByID retrieves a transaction by its primary key.
	FindByID(ctx context.Context, transactionID int64) (*model.Transaction, error)

	// FindOperationType retrieves an operation type by its numeric ID.
	// Returns an error if the operation type does not exist.
	FindOperationType(ctx context.Context, operationTypeID int64) (*model.OperationType, error)
}
