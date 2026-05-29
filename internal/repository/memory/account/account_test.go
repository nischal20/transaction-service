package account_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	memaccount "github.com/nischalpatel/transactions-api/internal/repository/memory/account"
	"github.com/nischalpatel/transactions-api/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountStore_Create_Success(t *testing.T) {
	store := memaccount.NewAccountStore()

	acc, err := store.Create(context.Background(), "12345678900")
	require.NoError(t, err)
	assert.Greater(t, acc.AccountID, int64(0))
	assert.Equal(t, "12345678900", acc.DocumentNumber)
}

func TestAccountStore_Create_UniqueIDs(t *testing.T) {
	store := memaccount.NewAccountStore()

	acc1, err := store.Create(context.Background(), "11111111111")
	require.NoError(t, err)
	acc2, err := store.Create(context.Background(), "22222222222")
	require.NoError(t, err)

	assert.NotEqual(t, acc1.AccountID, acc2.AccountID)
}

func TestAccountStore_Create_DuplicateDocument(t *testing.T) {
	store := memaccount.NewAccountStore()

	_, err := store.Create(context.Background(), "12345678900")
	require.NoError(t, err)

	_, err = store.Create(context.Background(), "12345678900")
	assert.EqualError(t, err, "document_number already exists")
}

func TestAccountStore_FindByID_Found(t *testing.T) {
	store := memaccount.NewAccountStore()

	created, err := store.Create(context.Background(), "12345678900")
	require.NoError(t, err)

	found, err := store.FindByID(context.Background(), created.AccountID)
	require.NoError(t, err)
	assert.Equal(t, created.AccountID, found.AccountID)
	assert.Equal(t, created.DocumentNumber, found.DocumentNumber)
}

func TestAccountStore_FindByID_NotFound(t *testing.T) {
	store := memaccount.NewAccountStore()

	_, err := store.FindByID(context.Background(), 999)
	assert.True(t, errors.Is(err, repository.ErrNotFound))
}

func TestAccountStore_FindByID_NotFound_EmptyStore(t *testing.T) {
	store := memaccount.NewAccountStore()

	_, err := store.FindByID(context.Background(), 1)
	assert.Error(t, err)
}

func TestAccountStore_Create_ConcurrentSafe(t *testing.T) {
	store := memaccount.NewAccountStore()
	const n = 100

	var wg sync.WaitGroup
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = store.Create(context.Background(), fmt.Sprintf("%011d", i))
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d failed", i)
	}
}

func TestAccountStore_Create_ConcurrentDuplicate(t *testing.T) {
	store := memaccount.NewAccountStore()
	const n = 50

	var wg sync.WaitGroup
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = store.Create(context.Background(), "same-document")
		}(i)
	}
	wg.Wait()

	successes := 0
	for _, err := range errs {
		if err == nil {
			successes++
		}
	}
	assert.Equal(t, 1, successes, "exactly one goroutine should win the duplicate race")
}
