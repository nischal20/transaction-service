package account

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"

	"github.com/nischalpatel/transactions-api/internal/model"
	accountrepo "github.com/nischalpatel/transactions-api/internal/repository/account"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// AccountStore is a thread-safe in-memory implementation of AccountRepository.
type AccountStore struct {
	mu       sync.Mutex
	accounts map[int64]*model.Account
	docIndex map[string]struct{}
	counter  atomic.Int64
}

func NewAccountStore() *AccountStore {
	return &AccountStore{
		accounts: make(map[int64]*model.Account),
		docIndex: make(map[string]struct{}),
	}
}

func (s *AccountStore) Create(ctx context.Context, _ *sql.Tx, documentNumber string) (*model.Account, error) {
	utils.Logf(ctx, "repo[memory]: insert account document_number=%q", documentNumber)
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.docIndex[documentNumber]; exists {
		utils.Logf(ctx, "repo[memory]: insert account failed: duplicate document_number=%q", documentNumber)
		return nil, accountrepo.ErrDuplicateDocument
	}

	id := s.counter.Add(1)
	acc := &model.Account{
		AccountID:      id,
		DocumentNumber: documentNumber,
	}
	s.accounts[id] = acc
	s.docIndex[documentNumber] = struct{}{}
	utils.Logf(ctx, "repo[memory]: insert account ok account_id=%d", id)
	return acc, nil
}

func (s *AccountStore) FindByID(ctx context.Context, accountID int64) (*model.Account, error) {
	utils.Logf(ctx, "repo[memory]: find account account_id=%d", accountID)
	s.mu.Lock()
	acc, ok := s.accounts[accountID]
	s.mu.Unlock()

	if !ok {
		utils.Logf(ctx, "repo[memory]: find account not found account_id=%d", accountID)
		return nil, accountrepo.ErrNotFound
	}
	utils.Logf(ctx, "repo[memory]: find account ok document_number=%q", acc.DocumentNumber)
	return acc, nil
}
