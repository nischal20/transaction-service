package transaction_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/audit"
	"github.com/nischalpatel/transactions-api/internal/handler/account"
	"github.com/nischalpatel/transactions-api/internal/handler/transaction"
	"github.com/nischalpatel/transactions-api/internal/model"
	memaccount "github.com/nischalpatel/transactions-api/internal/repository/memory/account"
	memtransaction "github.com/nischalpatel/transactions-api/internal/repository/memory/transaction"
	svcaccount "github.com/nischalpatel/transactions-api/internal/service/account"
	svctransaction "github.com/nischalpatel/transactions-api/internal/service/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubTransactionSvc lets tests control exactly what the service returns.
type stubTransactionSvc struct {
	createErr   error
	transaction *model.Transaction
}

func (s *stubTransactionSvc) CreateTransaction(_ context.Context, _, _ int64, _ float64) (*model.Transaction, error) {
	return s.transaction, s.createErr
}

func newTransactionRouterWithStub(svc svctransaction.TransactionServicer) http.Handler {
	h := transaction.NewTransactionHandler(svc)
	r := chi.NewRouter()
	r.Post("/transactions", h.CreateTransaction)
	return r
}

func newTransactionRouter() http.Handler {
	accStore := memaccount.NewAccountStore()
	txStore := memtransaction.NewTransactionStore()
	accSvc := svcaccount.NewAccountService(accStore, audit.NoopLogger{}, nil)
	txSvc := svctransaction.NewTransactionService(txStore, accStore, audit.NoopLogger{}, nil)

	accHandler := account.NewAccountHandler(accSvc)
	txHandler := transaction.NewTransactionHandler(txSvc)

	r := chi.NewRouter()
	r.Post("/accounts", accHandler.CreateAccount)
	r.Post("/transactions", txHandler.CreateTransaction)
	return r
}

func createAccount(t *testing.T, router http.Handler, docNumber string) model.Account {
	t.Helper()
	body := `{"document_number":"` + docNumber + `"}`
	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code)

	var acc model.Account
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &acc))
	return acc
}

func TestCreateTransaction_Purchase(t *testing.T) {
	router := newTransactionRouter()
	acc := createAccount(t, router, "11111111111")

	payload, _ := json.Marshal(map[string]any{
		"account_id":        acc.AccountID,
		"operation_type_id": model.OperationNormalPurchase,
		"amount":            123.45,
	})
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var tx model.Transaction
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tx))
	assert.Equal(t, -123.45, tx.Amount)
	assert.Equal(t, "debit", tx.Type)
}

func TestCreateTransaction_CreditVoucher(t *testing.T) {
	router := newTransactionRouter()
	acc := createAccount(t, router, "22222222222")

	payload, _ := json.Marshal(map[string]any{
		"account_id":        acc.AccountID,
		"operation_type_id": model.OperationCreditVoucher,
		"amount":            60.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var tx model.Transaction
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tx))
	assert.Equal(t, 60.0, tx.Amount)
	assert.Equal(t, "credit", tx.Type)
}

func TestCreateTransaction_AccountNotFound(t *testing.T) {
	router := newTransactionRouter()

	payload, _ := json.Marshal(map[string]any{
		"account_id":        999,
		"operation_type_id": model.OperationNormalPurchase,
		"amount":            50.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateTransaction_InvalidOperationType(t *testing.T) {
	router := newTransactionRouter()
	acc := createAccount(t, router, "33333333333")

	payload, _ := json.Marshal(map[string]any{
		"account_id":        acc.AccountID,
		"operation_type_id": 99,
		"amount":            50.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateTransaction_MalformedJSON(t *testing.T) {
	router := newTransactionRouter()

	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBufferString(`{bad json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ── Stub-based tests ─────────────────────────────────────────────────────────

func TestCreateTransaction_ServiceInternalError_Returns500(t *testing.T) {
	router := newTransactionRouterWithStub(&stubTransactionSvc{
		createErr: errors.New("storage failure"),
	})

	payload, _ := json.Marshal(map[string]any{
		"account_id": 1, "operation_type_id": 1, "amount": 50.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCreateTransaction_ServiceNotFound_Returns422(t *testing.T) {
	router := newTransactionRouterWithStub(&stubTransactionSvc{
		createErr: apperr.NotFound("account not found"),
	})

	payload, _ := json.Marshal(map[string]any{
		"account_id": 1, "operation_type_id": 1, "amount": 50.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestCreateTransaction_ServiceValidation_Returns400(t *testing.T) {
	router := newTransactionRouterWithStub(&stubTransactionSvc{
		createErr: apperr.Validation("invalid account_id"),
	})

	payload, _ := json.Marshal(map[string]any{
		"account_id": 1, "operation_type_id": 1, "amount": 50.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateTransaction_NegativeAmount_Rejected(t *testing.T) {
	router := newTransactionRouter()
	acc := createAccount(t, router, "44444444444")

	payload, _ := json.Marshal(map[string]any{
		"account_id":        acc.AccountID,
		"operation_type_id": model.OperationNormalPurchase,
		"amount":            -100.0,
	})
	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
