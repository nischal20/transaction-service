package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/nischalpatel/transactions-api/internal/model"
	"github.com/nischalpatel/transactions-api/internal/repository"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// AccountStore is a PostgreSQL-backed implementation of AccountRepository.
type AccountStore struct {
	db *sql.DB
}

func NewAccountStore(db *sql.DB) *AccountStore {
	return &AccountStore{db: db}
}

func (s *AccountStore) Create(ctx context.Context, documentNumber string) (*model.Account, error) {
	utils.Logf(ctx, "repo[postgres]: insert account document_number=%q", documentNumber)
	var acc model.Account
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO accounts (document_number) VALUES ($1) RETURNING account_id, document_number`,
		documentNumber,
	).Scan(&acc.AccountID, &acc.DocumentNumber)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			utils.Logf(ctx, "repo[postgres]: insert account failed: duplicate document_number=%q", documentNumber)
			return nil, repository.ErrDuplicateDocument
		}
		utils.Logf(ctx, "repo[postgres]: insert account error: %v", err)
		return nil, fmt.Errorf("insert account: %w", err)
	}
	utils.Logf(ctx, "repo[postgres]: insert account ok account_id=%d", acc.AccountID)
	return &acc, nil
}

func (s *AccountStore) FindByID(ctx context.Context, accountID int64) (*model.Account, error) {
	utils.Logf(ctx, "repo[postgres]: find account account_id=%d", accountID)
	var acc model.Account
	err := s.db.QueryRowContext(ctx,
		`SELECT account_id, document_number FROM accounts WHERE account_id = $1`,
		accountID,
	).Scan(&acc.AccountID, &acc.DocumentNumber)
	if errors.Is(err, sql.ErrNoRows) {
		utils.Logf(ctx, "repo[postgres]: find account not found account_id=%d", accountID)
		return nil, repository.ErrNotFound
	}
	if err != nil {
		utils.Logf(ctx, "repo[postgres]: find account error: %v", err)
		return nil, fmt.Errorf("query account: %w", err)
	}
	utils.Logf(ctx, "repo[postgres]: find account ok document_number=%q", acc.DocumentNumber)
	return &acc, nil
}
