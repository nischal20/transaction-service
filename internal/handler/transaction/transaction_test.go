package transaction_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/handler/transaction"
	"github.com/nischalpatel/transactions-api/internal/model"
)

// mockTransactionSvc satisfies svctransaction.TransactionServicer.
type mockTransactionSvc struct {
	createFn func(ctx context.Context, accountID, operationTypeID int64, amount int64, idemKey, idemHash string) (*model.Transaction, error)
}

func (m *mockTransactionSvc) CreateTransaction(ctx context.Context, accountID, operationTypeID int64, amount int64, idemKey, idemHash string) (*model.Transaction, error) {
	return m.createFn(ctx, accountID, operationTypeID, amount, idemKey, idemHash)
}

func fakeTransaction() *model.Transaction {
	return &model.Transaction{
		TransactionID:   42,
		AccountID:       1,
		OperationTypeID: 1,
		Amount:          -10000,
		Type:            "debit",
		EventDate:       time.Now(),
		CreatedAt:       time.Now(),
	}
}

const validBody = `{"account_id":1,"operation_type_id":1,"amount":100.00}`

func newReq(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Idempotency-Key", "test-idem-key-001")
	return req
}

// ---- CreateTransaction -----------------------------------------------------

func TestCreateTransaction_Success(t *testing.T) {
	svc := &mockTransactionSvc{createFn: func(_ context.Context, _, _ int64, _ int64, _, _ string) (*model.Transaction, error) {
		return fakeTransaction(), nil
	}}
	h := transaction.NewTransactionHandler(svc)

	w := httptest.NewRecorder()
	h.CreateTransaction(w, newReq(validBody))

	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, float64(42), resp["transaction_id"])
	assert.Equal(t, float64(1), resp["account_id"])
	assert.Equal(t, "debit", resp["type"])
	assert.Equal(t, -100.0, resp["amount"]) // -10000 paise → -100.00 rupees
}

func TestCreateTransaction_MissingIdempotencyKey(t *testing.T) {
	h := transaction.NewTransactionHandler(&mockTransactionSvc{})

	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader(validBody))
	// deliberately omit X-Idempotency-Key
	w := httptest.NewRecorder()

	h.CreateTransaction(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertErrorField(t, w, "X-Idempotency-Key header is required")
}

func TestCreateTransaction_BodyReadError(t *testing.T) {
	h := transaction.NewTransactionHandler(&mockTransactionSvc{})

	req := httptest.NewRequest(http.MethodPost, "/transactions", &errReader{err: errors.New("read failure")})
	req.Header.Set("X-Idempotency-Key", "key-001")
	w := httptest.NewRecorder()

	h.CreateTransaction(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assertErrorField(t, w, "failed to read request body")
}

// errReader is an io.Reader that always returns an error.
type errReader struct{ err error }

func (e *errReader) Read(_ []byte) (int, error) { return 0, e.err }

func TestCreateTransaction_InvalidJSON(t *testing.T) {
	h := transaction.NewTransactionHandler(&mockTransactionSvc{})

	req := httptest.NewRequest(http.MethodPost, "/transactions", strings.NewReader("not-json"))
	req.Header.Set("X-Idempotency-Key", "key-001")
	w := httptest.NewRecorder()

	h.CreateTransaction(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateTransaction_ValidationError(t *testing.T) {
	svc := &mockTransactionSvc{createFn: func(_ context.Context, _, _ int64, _ int64, _, _ string) (*model.Transaction, error) {
		return nil, apperr.Validation("amount must be greater than zero")
	}}
	h := transaction.NewTransactionHandler(svc)

	w := httptest.NewRecorder()
	h.CreateTransaction(w, newReq(validBody))

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateTransaction_AccountNotFound(t *testing.T) {
	svc := &mockTransactionSvc{createFn: func(_ context.Context, _, _ int64, _ int64, _, _ string) (*model.Transaction, error) {
		return nil, apperr.NotFound("account not found")
	}}
	h := transaction.NewTransactionHandler(svc)

	w := httptest.NewRecorder()
	h.CreateTransaction(w, newReq(validBody))

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateTransaction_IdempotencyConflict(t *testing.T) {
	svc := &mockTransactionSvc{createFn: func(_ context.Context, _, _ int64, _ int64, _, _ string) (*model.Transaction, error) {
		return nil, apperr.Conflict("idempotency key reused with a different request body")
	}}
	h := transaction.NewTransactionHandler(svc)

	w := httptest.NewRecorder()
	h.CreateTransaction(w, newReq(validBody))

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestCreateTransaction_InternalError(t *testing.T) {
	svc := &mockTransactionSvc{createFn: func(_ context.Context, _, _ int64, _ int64, _, _ string) (*model.Transaction, error) {
		return nil, errors.New("unexpected db failure")
	}}
	h := transaction.NewTransactionHandler(svc)

	w := httptest.NewRecorder()
	h.CreateTransaction(w, newReq(validBody))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestCreateTransaction_AmountConvertedToPaise(t *testing.T) {
	var capturedAmount int64
	svc := &mockTransactionSvc{createFn: func(_ context.Context, _, _ int64, amount int64, _, _ string) (*model.Transaction, error) {
		capturedAmount = amount
		return fakeTransaction(), nil
	}}
	h := transaction.NewTransactionHandler(svc)

	// 123.45 rupees → 12345 paise
	body := `{"account_id":1,"operation_type_id":1,"amount":123.45}`
	w := httptest.NewRecorder()
	h.CreateTransaction(w, newReq(body))

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, int64(12345), capturedAmount)
}

// assertErrorField checks that the JSON response has an "error" key matching msg.
func assertErrorField(t *testing.T, w *httptest.ResponseRecorder, msg string) {
	t.Helper()
	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, msg, resp["error"])
}
