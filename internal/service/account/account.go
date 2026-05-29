package account

import (
	"context"
	"errors"
	"strings"

	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/audit"
	"github.com/nischalpatel/transactions-api/internal/model"
	"github.com/nischalpatel/transactions-api/internal/repository"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// AccountServicer is the interface handlers depend on for account operations.
// Handlers never import the concrete AccountService — they depend on this
// interface, which allows stubs to be injected in tests without a real store.
type AccountServicer interface {
	// CreateAccount validates documentNumber and persists a new account.
	// Returns apperr.ErrValidation if empty, apperr.ErrConflict if already registered.
	CreateAccount(ctx context.Context, documentNumber string) (*model.Account, error)

	// GetAccount retrieves an account by its numeric ID.
	// Returns apperr.ErrValidation for invalid IDs, apperr.ErrNotFound if absent.
	GetAccount(ctx context.Context, accountID int64) (*model.Account, error)
}

// AccountService handles business logic for accounts.
type AccountService struct {
	repo    repository.AccountRepository
	auditor audit.Logger
}

func NewAccountService(repo repository.AccountRepository, auditor audit.Logger) *AccountService {
	return &AccountService{repo: repo, auditor: auditor}
}

// CreateAccount validates input and persists a new account.
func (s *AccountService) CreateAccount(ctx context.Context, documentNumber string) (*model.Account, error) {
	utils.Logf(ctx, "service: create account: document_number=%q", documentNumber)
	documentNumber = strings.TrimSpace(documentNumber)
	if documentNumber == "" {
		utils.Logf(ctx, "service: create account: validation failed: document_number is empty")
		return nil, apperr.Validation("document_number is required")
	}
	acc, err := s.repo.Create(ctx, documentNumber)
	if err != nil {
		utils.Logf(ctx, "service: create account: repo error: %v", err)
		if errors.Is(err, repository.ErrDuplicateDocument) {
			return nil, apperr.Conflict(err.Error())
		}
		return nil, err
	}
	utils.Logf(ctx, "service: create account: created account_id=%d", acc.AccountID)
	if err := s.auditor.Log(ctx, audit.Entry{
		EventType:  audit.EventAccountCreated,
		Resource:   "account",
		ResourceID: acc.AccountID,
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
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperr.NotFound("account not found")
		}
		return nil, err
	}
	utils.Logf(ctx, "service: get account: found document_number=%q", acc.DocumentNumber)
	return acc, nil
}
