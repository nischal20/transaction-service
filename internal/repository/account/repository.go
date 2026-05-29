package account

import (
	"context"
	"database/sql"
	"errors"

	"github.com/nischalpatel/transactions-api/internal/model"
)

var ErrDuplicateDocument = errors.New("document_number already exists")
var ErrNotFound = errors.New("not found")

// Repository defines persistence operations for accounts.
// Implementations must be safe for concurrent use.
type Repository interface {
	// Create persists a new account with the given document number.
	// tx must be a live transaction when the insert should be part of a larger
	// atomic operation; pass nil to run outside any transaction.
	// Returns ErrDuplicateDocument if the document number is already registered.
	Create(ctx context.Context, tx *sql.Tx, documentNumber string) (*model.Account, error)

	// FindByID retrieves an account by its numeric ID.
	// Returns ErrNotFound if no account with that ID exists.
	FindByID(ctx context.Context, accountID int64) (*model.Account, error)
}
