package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/audit"
	"github.com/nischalpatel/transactions-api/internal/model"
	accountrepo "github.com/nischalpatel/transactions-api/internal/repository/account"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// AccountServicer is the interface handlers depend on for account operations.
type AccountServicer interface {
	CreateAccount(ctx context.Context, documentNumber string) (*model.Account, error)
	GetAccount(ctx context.Context, accountID int64) (*model.Account, error)
}

// AccountService handles business logic for accounts.
type AccountService struct {
	repo    accountrepo.Repository
	auditor audit.Logger
	db      *sql.DB // non-nil in postgres mode; used to begin db transactions
}

func NewAccountService(repo accountrepo.Repository, auditor audit.Logger, db *sql.DB) *AccountService {
	return &AccountService{repo: repo, auditor: auditor, db: db}
}

// CreateAccount validates input and persists a new account.
//
// In PostgreSQL mode the account row and its audit entry are written inside a
// single database transaction — both commit or both roll back.
func (s *AccountService) CreateAccount(ctx context.Context, documentNumber string) (*model.Account, error) {
	utils.Logf(ctx, "service: create account: document_number=%q", documentNumber)
	documentNumber = strings.TrimSpace(documentNumber)
	if documentNumber == "" {
		utils.Logf(ctx, "service: create account: validation failed: document_number is empty")
		return nil, apperr.Validation("document_number is required")
	}

	if s.db != nil {
		return s.createWithDBTx(ctx, documentNumber)
	}
	return s.createInMemory(ctx, documentNumber)
}

func (s *AccountService) createWithDBTx(ctx context.Context, documentNumber string) (*model.Account, error) {
	utils.Logf(ctx, "service: create account: begin db tx")
	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin db tx: %w", err)
	}

	acc, err := s.repo.Create(ctx, sqlTx, documentNumber)
	if err != nil {
		_ = sqlTx.Rollback()
		utils.Logf(ctx, "service: create account: rollback db tx: repo error: %v", err)
		if errors.Is(err, accountrepo.ErrDuplicateDocument) {
			return nil, apperr.Conflict(err.Error())
		}
		return nil, err
	}
	utils.Logf(ctx, "service: create account: created account_id=%d", acc.AccountID)

	if err := s.auditor.Log(ctx, sqlTx, audit.Entry{
		EventType:  audit.EventAccountCreated,
		Resource:   "account",
		ResourceID: acc.AccountID,
		Actor:      utils.ActorFromCtx(ctx),
		RequestID:  utils.RequestIDFromCtx(ctx),
	}); err != nil {
		_ = sqlTx.Rollback()
		utils.Logf(ctx, "service: create account: rollback db tx: audit error: %v", err)
		return nil, err
	}

	if err := sqlTx.Commit(); err != nil {
		utils.Logf(ctx, "service: create account: commit failed: %v", err)
		return nil, fmt.Errorf("commit db tx: %w", err)
	}
	utils.Logf(ctx, "service: create account: committed db tx account_id=%d", acc.AccountID)
	return acc, nil
}

func (s *AccountService) createInMemory(ctx context.Context, documentNumber string) (*model.Account, error) {
	acc, err := s.repo.Create(ctx, nil, documentNumber)
	if err != nil {
		utils.Logf(ctx, "service: create account: repo error: %v", err)
		if errors.Is(err, accountrepo.ErrDuplicateDocument) {
			return nil, apperr.Conflict(err.Error())
		}
		return nil, err
	}
	utils.Logf(ctx, "service: create account: created account_id=%d", acc.AccountID)
	if err := s.auditor.Log(ctx, nil, audit.Entry{
		EventType:  audit.EventAccountCreated,
		Resource:   "account",
		ResourceID: acc.AccountID,
		Actor:      utils.ActorFromCtx(ctx),
		RequestID:  utils.RequestIDFromCtx(ctx),
	}); err != nil {
		utils.Logf(ctx, "service: create account: audit log warning (non-fatal): %v", err)
	}
	return acc, nil
}

// GetAccount retrieves an account by ID.
func (s *AccountService) GetAccount(ctx context.Context, accountID int64) (*model.Account, error) {
	utils.Logf(ctx, "service: get account: account_id=%d", accountID)
	if accountID <= 0 {
		utils.Logf(ctx, "service: get account: validation failed: invalid account_id=%d", accountID)
		return nil, apperr.Validation("invalid account_id")
	}
	acc, err := s.repo.FindByID(ctx, accountID)
	if err != nil {
		utils.Logf(ctx, "service: get account: repo error: %v", err)
		if errors.Is(err, accountrepo.ErrNotFound) {
			return nil, apperr.NotFound("account not found")
		}
		return nil, err
	}
	utils.Logf(ctx, "service: get account: found document_number=%q", acc.DocumentNumber)
	return acc, nil
}
