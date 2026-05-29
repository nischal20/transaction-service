package transaction

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nischalpatel/transactions-api/internal/model"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// TransactionStore is a thread-safe in-memory implementation of TransactionRepository.
type TransactionStore struct {
	mu             sync.RWMutex
	transactions   map[int64]*model.Transaction
	operationTypes map[int64]*model.OperationType
	counter        atomic.Int64
}

func NewTransactionStore() *TransactionStore {
	s := &TransactionStore{
		transactions:   make(map[int64]*model.Transaction),
		operationTypes: make(map[int64]*model.OperationType),
	}
	for _, ot := range model.SeedOperationTypes() {
		ot := ot
		s.operationTypes[ot.OperationTypeID] = &ot
	}
	return s
}

func (s *TransactionStore) Create(ctx context.Context, accountID, operationTypeID int64, amount float64, txType string) (*model.Transaction, error) {
	utils.Logf(ctx, "repo[memory]: insert transaction account_id=%d op_type=%d amount=%.2f", accountID, operationTypeID, amount)
	id := s.counter.Add(1)
	tx := &model.Transaction{
		TransactionID:   id,
		AccountID:       accountID,
		OperationTypeID: operationTypeID,
		Amount:          amount,
		Type:            txType,
		EventDate:       time.Now().UTC(),
	}

	s.mu.Lock()
	s.transactions[id] = tx
	s.mu.Unlock()

	utils.Logf(ctx, "repo[memory]: insert transaction ok transaction_id=%d", id)
	return tx, nil
}

func (s *TransactionStore) FindOperationType(ctx context.Context, operationTypeID int64) (*model.OperationType, error) {
	utils.Logf(ctx, "repo[memory]: find operation_type op_type=%d", operationTypeID)
	s.mu.RLock()
	ot, ok := s.operationTypes[operationTypeID]
	s.mu.RUnlock()

	if !ok {
		utils.Logf(ctx, "repo[memory]: find operation_type not found op_type=%d", operationTypeID)
		return nil, errors.New("operation type not found")
	}
	utils.Logf(ctx, "repo[memory]: find operation_type ok description=%q", ot.Description)
	return ot, nil
}
