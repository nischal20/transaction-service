package account_test

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
	"github.com/nischalpatel/transactions-api/internal/handler/account"
	"github.com/nischalpatel/transactions-api/internal/model"
	memaccount "github.com/nischalpatel/transactions-api/internal/repository/memory/account"
	svcaccount "github.com/nischalpatel/transactions-api/internal/service/account"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAccountSvc lets tests control exactly what the service returns.
type stubAccountSvc struct {
	createErr error
	getErr    error
	account   *model.Account
}

func (s *stubAccountSvc) CreateAccount(_ context.Context, _ string) (*model.Account, error) {
	return s.account, s.createErr
}

func (s *stubAccountSvc) GetAccount(_ context.Context, _ int64) (*model.Account, error) {
	return s.account, s.getErr
}

func newAccountRouterWithSvc(svc svcaccount.AccountServicer) http.Handler {
	h := account.NewAccountHandler(svc)
	r := chi.NewRouter()
	r.Post("/accounts", h.CreateAccount)
	r.Get("/accounts/{accountId}", h.GetAccount)
	return r
}

func newAccountRouter() http.Handler {
	accStore := memaccount.NewAccountStore()
	accSvc := svcaccount.NewAccountService(accStore)
	h := account.NewAccountHandler(accSvc)

	r := chi.NewRouter()
	r.Post("/accounts", h.CreateAccount)
	r.Get("/accounts/{accountId}", h.GetAccount)
	return r
}

func TestCreateAccount_Created(t *testing.T) {
	router := newAccountRouter()

	body := `{"document_number":"12345678900"}`
	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var acc model.Account
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &acc))
	assert.Equal(t, "12345678900", acc.DocumentNumber)
	assert.Greater(t, acc.AccountID, int64(0))
}

func TestCreateAccount_MissingDocument(t *testing.T) {
	router := newAccountRouter()

	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(`{"document_number":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateAccount_MalformedJSON(t *testing.T) {
	router := newAccountRouter()

	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(`{bad json`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCreateAccount_DuplicateDocument(t *testing.T) {
	router := newAccountRouter()

	body := `{"document_number":"12345678900"}`
	req1 := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(body))
	rec1 := httptest.NewRecorder()
	router.ServeHTTP(rec1, req1)
	require.Equal(t, http.StatusCreated, rec1.Code)

	req2 := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(body))
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)

	assert.Equal(t, http.StatusConflict, rec2.Code)
}

func TestGetAccount_Found(t *testing.T) {
	router := newAccountRouter()

	createReq := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(`{"document_number":"99999999999"}`))
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusCreated, createRec.Code)

	var created model.Account
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &created))

	getReq := httptest.NewRequest(http.MethodGet, "/accounts/1", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)

	assert.Equal(t, http.StatusOK, getRec.Code)
}

func TestGetAccount_NotFound(t *testing.T) {
	router := newAccountRouter()

	req := httptest.NewRequest(http.MethodGet, "/accounts/999", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetAccount_NonNumericID(t *testing.T) {
	router := newAccountRouter()

	req := httptest.NewRequest(http.MethodGet, "/accounts/abc", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ── Stub-based tests — isolate the handler from the real service ─────────────

func TestCreateAccount_ServiceInternalError_Returns500(t *testing.T) {
	router := newAccountRouterWithSvc(&stubAccountSvc{
		createErr: errors.New("storage failure"),
	})

	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(`{"document_number":"12345678900"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestGetAccount_ServiceInternalError_Returns500(t *testing.T) {
	router := newAccountRouterWithSvc(&stubAccountSvc{
		getErr: errors.New("storage failure"),
	})

	req := httptest.NewRequest(http.MethodGet, "/accounts/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestCreateAccount_ServiceConflict_Returns409(t *testing.T) {
	router := newAccountRouterWithSvc(&stubAccountSvc{
		createErr: apperr.Conflict("document_number already exists"),
	})

	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(`{"document_number":"12345678900"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestGetAccount_ServiceNotFound_Returns404(t *testing.T) {
	router := newAccountRouterWithSvc(&stubAccountSvc{
		getErr: apperr.NotFound("account not found"),
	})

	req := httptest.NewRequest(http.MethodGet, "/accounts/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
