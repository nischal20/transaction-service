package transaction_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/audit"
	"github.com/nischalpatel/transactions-api/internal/idempotency"
	"github.com/nischalpatel/transactions-api/internal/model"
	accountrepo "github.com/nischalpatel/transactions-api/internal/repository/account"
	svctransaction "github.com/nischalpatel/transactions-api/internal/service/transaction"
)

// ---- mocks -----------------------------------------------------------------

type mockTxRepo struct {
	createFn           func(ctx context.Context, tx *sql.Tx, accountID, opTypeID, amount int64, txType string) (*model.Transaction, error)
	findByIDFn         func(ctx context.Context, transactionID int64) (*model.Transaction, error)
	findOperationTypFn func(ctx context.Context, operationTypeID int64) (*model.OperationType, error)
}

func (m *mockTxRepo) Create(ctx context.Context, tx *sql.Tx, accountID, opTypeID, amount int64, txType string) (*model.Transaction, error) {
	return m.createFn(ctx, tx, accountID, opTypeID, amount, txType)
}
func (m *mockTxRepo) FindByID(ctx context.Context, id int64) (*model.Transaction, error) {
	return m.findByIDFn(ctx, id)
}
func (m *mockTxRepo) FindOperationType(ctx context.Context, id int64) (*model.OperationType, error) {
	return m.findOperationTypFn(ctx, id)
}

type mockAccRepo struct {
	findByIDFn func(ctx context.Context, accountID int64) (*model.Account, error)
}

func (m *mockAccRepo) Create(_ context.Context, _ *sql.Tx, _ string) (*model.Account, error) {
	return nil, errors.New("not implemented")
}
func (m *mockAccRepo) FindByID(ctx context.Context, id int64) (*model.Account, error) {
	return m.findByIDFn(ctx, id)
}

type mockAuditor struct {
	logFn func(ctx context.Context, tx *sql.Tx, entry audit.Entry) error
}

func (m *mockAuditor) Log(ctx context.Context, tx *sql.Tx, entry audit.Entry) error {
	if m.logFn != nil {
		return m.logFn(ctx, tx, entry)
	}
	return nil
}

type mockIdem struct {
	findFn func(ctx context.Context, key string) (*idempotency.Record, error)
	saveFn func(ctx context.Context, tx *sql.Tx, key, hash string, txID int64) error
}

func (m *mockIdem) Find(ctx context.Context, key string) (*idempotency.Record, error) {
	return m.findFn(ctx, key)
}
func (m *mockIdem) Save(ctx context.Context, tx *sql.Tx, key, hash string, txID int64) error {
	return m.saveFn(ctx, tx, key, hash, txID)
}

// ---- helpers ---------------------------------------------------------------

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func fakeTx() *model.Transaction {
	return &model.Transaction{
		TransactionID: 42, AccountID: 1, OperationTypeID: 1,
		Amount: -10000, Type: "debit", EventDate: time.Now(), CreatedAt: time.Now(),
	}
}

// okDeps returns mocks wired for a clean happy-path insert (no idempotency hit).
func okDeps() (*mockTxRepo, *mockAccRepo, *mockAuditor, *mockIdem) {
	txRepo := &mockTxRepo{
		createFn: func(_ context.Context, _ *sql.Tx, _, _, _ int64, _ string) (*model.Transaction, error) {
			return fakeTx(), nil
		},
		findOperationTypFn: func(_ context.Context, _ int64) (*model.OperationType, error) {
			return &model.OperationType{OperationTypeID: 1, Description: "Normal Purchase"}, nil
		},
	}
	accRepo := &mockAccRepo{
		findByIDFn: func(_ context.Context, _ int64) (*model.Account, error) {
			return &model.Account{AccountID: 1}, nil
		},
	}
	auditor := &mockAuditor{}
	idem := &mockIdem{
		findFn: func(_ context.Context, _ string) (*idempotency.Record, error) { return nil, nil },
		saveFn: func(_ context.Context, _ *sql.Tx, _, _ string, _ int64) error { return nil },
	}
	return txRepo, accRepo, auditor, idem
}

func buildSvc(db *sql.DB, txRepo *mockTxRepo, accRepo *mockAccRepo, auditor *mockAuditor, idem *mockIdem) *svctransaction.TransactionService {
	return svctransaction.NewTransactionService(txRepo, accRepo, auditor, idem, db)
}

// ---- CreateTransaction: input validation -----------------------------------

func TestCreateTransaction_InvalidAccountID(t *testing.T) {
	db, _ := newMockDB(t)
	txRepo, accRepo, auditor, idem := okDeps()
	svc := buildSvc(db, txRepo, accRepo, auditor, idem)

	_, err := svc.CreateTransaction(context.Background(), 0, 1, 10000, "k1", "h1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrValidation))
}

func TestCreateTransaction_InvalidAmount(t *testing.T) {
	db, _ := newMockDB(t)
	txRepo, accRepo, auditor, idem := okDeps()
	svc := buildSvc(db, txRepo, accRepo, auditor, idem)

	_, err := svc.CreateTransaction(context.Background(), 1, 1, 0, "k1", "h1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrValidation))
}

// ---- CreateTransaction: idempotency ----------------------------------------

func TestCreateTransaction_IdempotencyHit_SameHash(t *testing.T) {
	db, _ := newMockDB(t)
	txRepo, accRepo, auditor, idem := okDeps()

	existingTx := fakeTx()
	idem.findFn = func(_ context.Context, _ string) (*idempotency.Record, error) {
		return &idempotency.Record{Key: "k1", RequestHash: "h1", TransactionID: existingTx.TransactionID}, nil
	}
	txRepo.findByIDFn = func(_ context.Context, id int64) (*model.Transaction, error) {
		return existingTx, nil
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	tx, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.NoError(t, err)
	assert.Equal(t, existingTx.TransactionID, tx.TransactionID)
}

func TestCreateTransaction_IdempotencyHit_DifferentHash(t *testing.T) {
	db, _ := newMockDB(t)
	txRepo, accRepo, auditor, idem := okDeps()

	idem.findFn = func(_ context.Context, _ string) (*idempotency.Record, error) {
		return &idempotency.Record{Key: "k1", RequestHash: "original-hash", TransactionID: 1}, nil
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "different-hash")

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrConflict))
}

func TestCreateTransaction_IdempotencyLookupError(t *testing.T) {
	db, _ := newMockDB(t)
	txRepo, accRepo, auditor, idem := okDeps()

	idem.findFn = func(_ context.Context, _ string) (*idempotency.Record, error) {
		return nil, errors.New("redis timeout")
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.Error(t, err)
}

// ---- CreateTransaction: pre-insert lookups ---------------------------------

func TestCreateTransaction_AccountNotFound(t *testing.T) {
	db, _ := newMockDB(t)
	txRepo, accRepo, auditor, idem := okDeps()

	accRepo.findByIDFn = func(_ context.Context, _ int64) (*model.Account, error) {
		return nil, accountrepo.ErrNotFound
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrNotFound))
}

func TestCreateTransaction_OperationTypeNotFound(t *testing.T) {
	db, _ := newMockDB(t)
	txRepo, accRepo, auditor, idem := okDeps()

	txRepo.findOperationTypFn = func(_ context.Context, _ int64) (*model.OperationType, error) {
		return nil, errors.New("not found")
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, 99, 10000, "k1", "h1")

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrNotFound))
}

// ---- CreateTransaction: happy path -----------------------------------------

func TestCreateTransaction_DebitAmountNegated(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	txRepo, accRepo, auditor, idem := okDeps()

	var capturedAmount int64
	txRepo.createFn = func(_ context.Context, _ *sql.Tx, _, _ int64, amount int64, txType string) (*model.Transaction, error) {
		capturedAmount = amount
		return &model.Transaction{TransactionID: 1, Amount: amount, Type: txType}, nil
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, model.OperationNormalPurchase, 10000, "k1", "h1")

	require.NoError(t, err)
	assert.Equal(t, int64(-10000), capturedAmount, "debit amounts must be stored as negative")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_CreditAmountPositive(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	txRepo, accRepo, auditor, idem := okDeps()
	txRepo.findOperationTypFn = func(_ context.Context, _ int64) (*model.OperationType, error) {
		return &model.OperationType{OperationTypeID: model.OperationCreditVoucher, Description: "Credit Voucher"}, nil
	}

	var capturedAmount int64
	txRepo.createFn = func(_ context.Context, _ *sql.Tx, _, _ int64, amount int64, _ string) (*model.Transaction, error) {
		capturedAmount = amount
		return &model.Transaction{TransactionID: 2, Amount: amount, Type: "credit"}, nil
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, model.OperationCreditVoucher, 5000, "k2", "h2")

	require.NoError(t, err)
	assert.Equal(t, int64(5000), capturedAmount, "credit amounts must remain positive")
	require.NoError(t, mock.ExpectationsWereMet())
}

// ---- CreateTransaction: failure paths after BeginTx -------------------------

func TestCreateTransaction_TxRepoError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	txRepo, accRepo, auditor, idem := okDeps()
	txRepo.createFn = func(_ context.Context, _ *sql.Tx, _, _, _ int64, _ string) (*model.Transaction, error) {
		return nil, errors.New("insert failed")
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_AuditError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	txRepo, accRepo, auditor, idem := okDeps()
	auditor.logFn = func(_ context.Context, _ *sql.Tx, _ audit.Entry) error {
		return errors.New("audit failed")
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_BeginTxError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin().WillReturnError(errors.New("connection pool exhausted"))

	txRepo, accRepo, auditor, idem := okDeps()
	svc := buildSvc(db, txRepo, accRepo, auditor, idem)

	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "begin db tx")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_IdemSaveNonConflictError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	txRepo, accRepo, auditor, idem := okDeps()
	idem.saveFn = func(_ context.Context, _ *sql.Tx, _, _ string, _ int64) error {
		return errors.New("network timeout saving idempotency record")
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.Error(t, err)
	assert.NotErrorIs(t, err, apperr.ErrConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_IdemRaceRelookupError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	txRepo, accRepo, auditor, idem := okDeps()
	idem.saveFn = func(_ context.Context, _ *sql.Tx, _, _ string, _ int64) error {
		return idempotency.ErrKeyConflict
	}
	calls := 0
	idem.findFn = func(_ context.Context, _ string) (*idempotency.Record, error) {
		calls++
		if calls == 1 {
			return nil, nil
		}
		return nil, errors.New("db error on re-lookup")
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "idempotency re-lookup after conflict")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_IdemRaceRelookupRecordNil(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	txRepo, accRepo, auditor, idem := okDeps()
	idem.saveFn = func(_ context.Context, _ *sql.Tx, _, _ string, _ int64) error {
		return idempotency.ErrKeyConflict
	}
	calls := 0
	idem.findFn = func(_ context.Context, _ string) (*idempotency.Record, error) {
		calls++
		return nil, nil // both calls return no record
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "idempotency conflict but no record found")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_CommitError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit().WillReturnError(errors.New("commit timeout"))

	txRepo, accRepo, auditor, idem := okDeps()
	svc := buildSvc(db, txRepo, accRepo, auditor, idem)

	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit db tx")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTransaction_IdemHitFindByIDError(t *testing.T) {
	db, _ := newMockDB(t)
	txRepo, accRepo, auditor, idem := okDeps()

	idem.findFn = func(_ context.Context, _ string) (*idempotency.Record, error) {
		return &idempotency.Record{Key: "k1", RequestHash: "h1", TransactionID: 42}, nil
	}
	txRepo.findByIDFn = func(_ context.Context, _ int64) (*model.Transaction, error) {
		return nil, errors.New("transaction 42 not found")
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	_, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.Error(t, err)
}

func TestCreateTransaction_IdempotencyRace(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	txRepo, accRepo, auditor, idem := okDeps()

	existingTx := fakeTx()
	idem.saveFn = func(_ context.Context, _ *sql.Tx, _, _ string, _ int64) error {
		return idempotency.ErrKeyConflict
	}
	idem.findFn = func() func(context.Context, string) (*idempotency.Record, error) {
		calls := 0
		return func(_ context.Context, _ string) (*idempotency.Record, error) {
			calls++
			if calls == 1 {
				return nil, nil // first call: no hit, proceed with insert
			}
			return &idempotency.Record{Key: "k1", RequestHash: "h1", TransactionID: existingTx.TransactionID}, nil
		}
	}()
	txRepo.findByIDFn = func(_ context.Context, _ int64) (*model.Transaction, error) {
		return existingTx, nil
	}

	svc := buildSvc(db, txRepo, accRepo, auditor, idem)
	tx, err := svc.CreateTransaction(context.Background(), 1, 1, 10000, "k1", "h1")

	require.NoError(t, err)
	assert.Equal(t, existingTx.TransactionID, tx.TransactionID)
	require.NoError(t, mock.ExpectationsWereMet())
}
