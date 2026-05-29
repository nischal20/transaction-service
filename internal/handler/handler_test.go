package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nischalpatel/transactions-api/internal/handler"
	handlerAccount "github.com/nischalpatel/transactions-api/internal/handler/account"
	handlerTransaction "github.com/nischalpatel/transactions-api/internal/handler/transaction"
	memaccount "github.com/nischalpatel/transactions-api/internal/repository/memory/account"
	memtransaction "github.com/nischalpatel/transactions-api/internal/repository/memory/transaction"
	svcaccount "github.com/nischalpatel/transactions-api/internal/service/account"
	svctransaction "github.com/nischalpatel/transactions-api/internal/service/transaction"
	"github.com/stretchr/testify/assert"
)

func newTestRouter() http.Handler {
	accStore := memaccount.NewAccountStore()
	txStore := memtransaction.NewTransactionStore()
	accSvc := svcaccount.NewAccountService(accStore)
	txSvc := svctransaction.NewTransactionService(txStore, accStore)

	return handler.NewRouter(
		handlerAccount.NewAccountHandler(accSvc),
		handlerTransaction.NewTransactionHandler(txSvc),
	)
}

func TestGetHealth(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMethodNotAllowed_Accounts(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestMethodNotAllowed_AccountsById(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/accounts/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestMethodNotAllowed_Transactions(t *testing.T) {
	router := newTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/transactions", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}
