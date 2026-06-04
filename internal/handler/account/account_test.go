package account_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/handler/account"
	"github.com/nischalpatel/transactions-api/internal/model"
)

// mockAccountSvc satisfies svcaccount.AccountServicer.
type mockAccountSvc struct {
	createFn func(ctx context.Context, documentNumber string) (*model.Account, error)
	getFn    func(ctx context.Context, accountID int64) (*model.Account, error)
}

func (m *mockAccountSvc) CreateAccount(ctx context.Context, documentNumber string) (*model.Account, error) {
	return m.createFn(ctx, documentNumber)
}
func (m *mockAccountSvc) GetAccount(ctx context.Context, accountID int64) (*model.Account, error) {
	return m.getFn(ctx, accountID)
}

func fakeAccount() *model.Account {
	return &model.Account{AccountID: 1, DocumentNumber: "12345678900", CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

// ---- CreateAccount ---------------------------------------------------------

func TestCreateAccount_Success(t *testing.T) {
	svc := &mockAccountSvc{createFn: func(_ context.Context, _ string) (*model.Account, error) {
		return fakeAccount(), nil
	}}
	h := account.NewAccountHandler(svc)

	body := `{"document_number":"12345678900"}`
	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateAccount(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["account_id"])
	assert.Equal(t, "12345678900", resp["document_number"])
}

func TestCreateAccount_InvalidJSON(t *testing.T) {
	h := account.NewAccountHandler(&mockAccountSvc{})

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader("not-json"))
	w := httptest.NewRecorder()

	h.CreateAccount(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAccount_ValidationError(t *testing.T) {
	svc := &mockAccountSvc{createFn: func(_ context.Context, _ string) (*model.Account, error) {
		return nil, apperr.Validation("document_number is required")
	}}
	h := account.NewAccountHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(`{"document_number":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateAccount(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateAccount_Conflict(t *testing.T) {
	svc := &mockAccountSvc{createFn: func(_ context.Context, _ string) (*model.Account, error) {
		return nil, apperr.Conflict("document_number already exists")
	}}
	h := account.NewAccountHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(`{"document_number":"12345678900"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateAccount(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateAccount_InternalError(t *testing.T) {
	svc := &mockAccountSvc{createFn: func(_ context.Context, _ string) (*model.Account, error) {
		return nil, errors.New("unexpected db failure")
	}}
	h := account.NewAccountHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(`{"document_number":"12345678900"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateAccount(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ---- GetAccount ------------------------------------------------------------

// withChiParam injects a chi URL parameter so handlers can call chi.URLParam.
func withChiParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestGetAccount_Success(t *testing.T) {
	svc := &mockAccountSvc{getFn: func(_ context.Context, _ int64) (*model.Account, error) {
		return fakeAccount(), nil
	}}
	h := account.NewAccountHandler(svc)

	req := withChiParam(httptest.NewRequest(http.MethodGet, "/accounts/1", nil), "accountId", "1")
	w := httptest.NewRecorder()

	h.GetAccount(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["account_id"])
	assert.Equal(t, "12345678900", resp["document_number"])
}

func TestGetAccount_InvalidID(t *testing.T) {
	h := account.NewAccountHandler(&mockAccountSvc{})

	req := withChiParam(httptest.NewRequest(http.MethodGet, "/accounts/abc", nil), "accountId", "abc")
	w := httptest.NewRecorder()

	h.GetAccount(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAccount_NotFound(t *testing.T) {
	svc := &mockAccountSvc{getFn: func(_ context.Context, _ int64) (*model.Account, error) {
		return nil, apperr.NotFound("account not found")
	}}
	h := account.NewAccountHandler(svc)

	req := withChiParam(httptest.NewRequest(http.MethodGet, "/accounts/999", nil), "accountId", "999")
	w := httptest.NewRecorder()

	h.GetAccount(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAccount_ValidationError(t *testing.T) {
	svc := &mockAccountSvc{getFn: func(_ context.Context, _ int64) (*model.Account, error) {
		return nil, apperr.Validation("invalid account_id")
	}}
	h := account.NewAccountHandler(svc)

	req := withChiParam(httptest.NewRequest(http.MethodGet, "/accounts/0", nil), "accountId", "0")
	w := httptest.NewRecorder()

	h.GetAccount(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAccount_InternalError(t *testing.T) {
	svc := &mockAccountSvc{getFn: func(_ context.Context, _ int64) (*model.Account, error) {
		return nil, errors.New("unexpected db failure")
	}}
	h := account.NewAccountHandler(svc)

	req := withChiParam(httptest.NewRequest(http.MethodGet, "/accounts/1", nil), "accountId", "1")
	w := httptest.NewRecorder()

	h.GetAccount(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
