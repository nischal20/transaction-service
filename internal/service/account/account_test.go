package account_test

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
	"github.com/nischalpatel/transactions-api/internal/model"
	accountrepo "github.com/nischalpatel/transactions-api/internal/repository/account"
	svcaccount "github.com/nischalpatel/transactions-api/internal/service/account"
)

// ---- mocks -----------------------------------------------------------------

type mockAccountRepo struct {
	createFn   func(ctx context.Context, tx *sql.Tx, documentNumber string) (*model.Account, error)
	findByIDFn func(ctx context.Context, accountID int64) (*model.Account, error)
}

func (m *mockAccountRepo) Create(ctx context.Context, tx *sql.Tx, documentNumber string) (*model.Account, error) {
	return m.createFn(ctx, tx, documentNumber)
}
func (m *mockAccountRepo) FindByID(ctx context.Context, accountID int64) (*model.Account, error) {
	return m.findByIDFn(ctx, accountID)
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

// ---- helpers ---------------------------------------------------------------

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func fakeAccount() *model.Account {
	return &model.Account{AccountID: 1, DocumentNumber: "12345678900", CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func okRepo() *mockAccountRepo {
	return &mockAccountRepo{
		createFn: func(_ context.Context, _ *sql.Tx, _ string) (*model.Account, error) {
			return fakeAccount(), nil
		},
	}
}

func okAuditor() *mockAuditor { return &mockAuditor{} }

// ---- CreateAccount ---------------------------------------------------------

func TestCreateAccount_Success(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	svc := svcaccount.NewAccountService(okRepo(), okAuditor(), db)
	acc, err := svc.CreateAccount(context.Background(), "12345678900")

	require.NoError(t, err)
	assert.Equal(t, int64(1), acc.AccountID)
	assert.Equal(t, "12345678900", acc.DocumentNumber)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAccount_EmptyDocumentNumber(t *testing.T) {
	db, _ := newMockDB(t)
	svc := svcaccount.NewAccountService(okRepo(), okAuditor(), db)

	_, err := svc.CreateAccount(context.Background(), "")

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrValidation))
}

func TestCreateAccount_WhitespaceDocumentNumber(t *testing.T) {
	db, _ := newMockDB(t)
	svc := svcaccount.NewAccountService(okRepo(), okAuditor(), db)

	_, err := svc.CreateAccount(context.Background(), "   ")

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrValidation))
}

func TestCreateAccount_DocumentNumberTooShort(t *testing.T) {
	db, _ := newMockDB(t)
	svc := svcaccount.NewAccountService(okRepo(), okAuditor(), db)

	_, err := svc.CreateAccount(context.Background(), "123456789") // 9 digits

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrValidation))
}

func TestCreateAccount_DocumentNumberTooLong(t *testing.T) {
	db, _ := newMockDB(t)
	svc := svcaccount.NewAccountService(okRepo(), okAuditor(), db)

	_, err := svc.CreateAccount(context.Background(), "123456789012345") // 15 digits

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrValidation))
}

func TestCreateAccount_DocumentNumberNonDigits(t *testing.T) {
	db, _ := newMockDB(t)
	svc := svcaccount.NewAccountService(okRepo(), okAuditor(), db)

	_, err := svc.CreateAccount(context.Background(), "1234567abc")

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrValidation))
}

func TestCreateAccount_DocumentNumberBoundaryMin(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	svc := svcaccount.NewAccountService(okRepo(), okAuditor(), db)
	_, err := svc.CreateAccount(context.Background(), "1234567890") // exactly 10

	require.NoError(t, err)
}

func TestCreateAccount_DocumentNumberBoundaryMax(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	svc := svcaccount.NewAccountService(okRepo(), okAuditor(), db)
	_, err := svc.CreateAccount(context.Background(), "12345678901234") // exactly 14

	require.NoError(t, err)
}

func TestCreateAccount_DuplicateDocument(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo := &mockAccountRepo{
		createFn: func(_ context.Context, _ *sql.Tx, _ string) (*model.Account, error) {
			return nil, accountrepo.ErrDuplicateDocument
		},
	}
	svc := svcaccount.NewAccountService(repo, okAuditor(), db)

	_, err := svc.CreateAccount(context.Background(), "12345678900")

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrConflict))
}

func TestCreateAccount_RepoError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	repo := &mockAccountRepo{
		createFn: func(_ context.Context, _ *sql.Tx, _ string) (*model.Account, error) {
			return nil, errors.New("db timeout")
		},
	}
	svc := svcaccount.NewAccountService(repo, okAuditor(), db)

	_, err := svc.CreateAccount(context.Background(), "12345678900")

	require.Error(t, err)
	assert.NotErrorIs(t, err, apperr.ErrConflict)
}

func TestCreateAccount_AuditError(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	auditor := &mockAuditor{
		logFn: func(_ context.Context, _ *sql.Tx, _ audit.Entry) error {
			return errors.New("audit write failed")
		},
	}
	svc := svcaccount.NewAccountService(okRepo(), auditor, db)

	_, err := svc.CreateAccount(context.Background(), "12345678900")

	require.Error(t, err)
}

// ---- GetAccount ------------------------------------------------------------

func TestGetAccount_Success(t *testing.T) {
	db, _ := newMockDB(t)
	repo := &mockAccountRepo{
		findByIDFn: func(_ context.Context, _ int64) (*model.Account, error) {
			return fakeAccount(), nil
		},
	}
	svc := svcaccount.NewAccountService(repo, okAuditor(), db)

	acc, err := svc.GetAccount(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, int64(1), acc.AccountID)
}

func TestGetAccount_ZeroID(t *testing.T) {
	db, _ := newMockDB(t)
	svc := svcaccount.NewAccountService(okRepo(), okAuditor(), db)

	_, err := svc.GetAccount(context.Background(), 0)

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrValidation))
}

func TestGetAccount_NegativeID(t *testing.T) {
	db, _ := newMockDB(t)
	svc := svcaccount.NewAccountService(okRepo(), okAuditor(), db)

	_, err := svc.GetAccount(context.Background(), -5)

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrValidation))
}

func TestGetAccount_NotFound(t *testing.T) {
	db, _ := newMockDB(t)
	repo := &mockAccountRepo{
		findByIDFn: func(_ context.Context, _ int64) (*model.Account, error) {
			return nil, accountrepo.ErrNotFound
		},
	}
	svc := svcaccount.NewAccountService(repo, okAuditor(), db)

	_, err := svc.GetAccount(context.Background(), 999)

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrNotFound))
}

func TestGetAccount_RepoError(t *testing.T) {
	db, _ := newMockDB(t)
	repo := &mockAccountRepo{
		findByIDFn: func(_ context.Context, _ int64) (*model.Account, error) {
			return nil, errors.New("db timeout")
		},
	}
	svc := svcaccount.NewAccountService(repo, okAuditor(), db)

	_, err := svc.GetAccount(context.Background(), 1)

	require.Error(t, err)
	assert.NotErrorIs(t, err, apperr.ErrNotFound)
}
