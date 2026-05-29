package account_test

import (
	"context"
	"testing"

	memaccount "github.com/nischalpatel/transactions-api/internal/repository/memory/account"
	svcaccount "github.com/nischalpatel/transactions-api/internal/service/account"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAccountService() *svcaccount.AccountService {
	return svcaccount.NewAccountService(memaccount.NewAccountStore())
}

func TestCreateAccount_Success(t *testing.T) {
	svc := newAccountService()

	acc, err := svc.CreateAccount(context.Background(), "12345678900")
	require.NoError(t, err)
	assert.Equal(t, "12345678900", acc.DocumentNumber)
	assert.Greater(t, acc.AccountID, int64(0))
}

func TestCreateAccount_EmptyDocument(t *testing.T) {
	svc := newAccountService()

	_, err := svc.CreateAccount(context.Background(), "   ")
	assert.EqualError(t, err, "document_number is required")
}

func TestCreateAccount_WhitespaceOnlyDocument(t *testing.T) {
	svc := newAccountService()

	_, err := svc.CreateAccount(context.Background(), "   \t  ")
	assert.EqualError(t, err, "document_number is required")
}

func TestCreateAccount_DuplicateDocument(t *testing.T) {
	svc := newAccountService()

	_, err := svc.CreateAccount(context.Background(), "12345678900")
	require.NoError(t, err)

	_, err = svc.CreateAccount(context.Background(), "12345678900")
	assert.EqualError(t, err, "document_number already exists")
}

func TestGetAccount_Found(t *testing.T) {
	svc := newAccountService()

	created, err := svc.CreateAccount(context.Background(), "99999999999")
	require.NoError(t, err)

	found, err := svc.GetAccount(context.Background(), created.AccountID)
	require.NoError(t, err)
	assert.Equal(t, created.AccountID, found.AccountID)
	assert.Equal(t, created.DocumentNumber, found.DocumentNumber)
}

func TestGetAccount_NotFound(t *testing.T) {
	svc := newAccountService()

	_, err := svc.GetAccount(context.Background(), 999)
	assert.Error(t, err)
}

func TestGetAccount_InvalidID(t *testing.T) {
	svc := newAccountService()

	_, err := svc.GetAccount(context.Background(), 0)
	assert.EqualError(t, err, "invalid account_id")
}

func TestGetAccount_NegativeID(t *testing.T) {
	svc := newAccountService()

	_, err := svc.GetAccount(context.Background(), -1)
	assert.EqualError(t, err, "invalid account_id")
}
