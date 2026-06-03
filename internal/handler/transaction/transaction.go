package transaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"

	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/dto"
	"github.com/nischalpatel/transactions-api/internal/idempotency"
	svctransaction "github.com/nischalpatel/transactions-api/internal/service/transaction"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// TransactionHandler holds HTTP handlers for transaction endpoints.
type TransactionHandler struct {
	svc         svctransaction.TransactionServicer
	idempotency idempotency.Repository
}

func NewTransactionHandler(svc svctransaction.TransactionServicer, idem idempotency.Repository) *TransactionHandler {
	return &TransactionHandler{svc: svc, idempotency: idem}
}

// CreateTransaction handles POST /transactions
//
//	@Summary		Create a transaction
//	@Description	Records a financial operation against an existing account.
//	@Description	Operation type IDs: 1=Normal Purchase, 2=Purchase with installments, 3=Withdrawal, 4=Credit Voucher.
//	@Description	Send a positive amount in rupees. The server applies the correct sign based on operation type.
//	@Description	Requires the X-Idempotency-Key header. Replaying the same key returns the original response.
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			X-Idempotency-Key	header		string						true	"Unique key to prevent duplicate submissions"
//	@Param			body				body		dto.CreateTransactionRequest	true	"Transaction payload"
//	@Success		201					{object}	dto.TransactionResponse
//	@Failure		400					{object}	dto.ErrorResponse
//	@Failure		405					{object}	dto.ErrorResponse
//	@Failure		422					{object}	dto.ErrorResponse
//	@Router			/transactions [post]
func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("X-Idempotency-Key")
	if idempotencyKey == "" {
		utils.WriteError(w, http.StatusBadRequest, "X-Idempotency-Key header is required")
		return
	}

	// Read body once — needed for both hashing and decoding.
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	sum := sha256.Sum256(rawBody)
	requestHash := hex.EncodeToString(sum[:])

	ctx := r.Context()

	// Check idempotency cache.
	if rec, err := h.idempotency.Find(ctx, idempotencyKey); err != nil {
		utils.Logf(ctx, "handler: idempotency lookup error: %v", err)
	} else if rec != nil {
		if rec.RequestHash != requestHash {
			utils.WriteError(w, http.StatusUnprocessableEntity, "idempotency key reused with a different request body")
			return
		}
		// Replay the original response exactly.
		utils.Logf(ctx, "handler: idempotency cache hit key=%s", idempotencyKey)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(rec.ResponseCode)
		_, _ = w.Write(rec.ResponseBody)
		return
	}

	var body dto.CreateTransactionRequest
	if err := json.NewDecoder(bytes.NewReader(rawBody)).Decode(&body); err != nil {
		utils.Logf(ctx, "handler: create transaction: decode error: %v", err)
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	amountPaise := int64(math.Round(body.Amount * 100))
	utils.Logf(ctx, "handler: create transaction: account_id=%d op_type=%d amount_rs=%.2f amount_paise=%d key=%s",
		body.AccountID, body.OperationTypeID, body.Amount, amountPaise, idempotencyKey)

	ctx = utils.WithActor(ctx, r.Header.Get("X-User-ID"))
	tx, err := h.svc.CreateTransaction(ctx, body.AccountID, body.OperationTypeID, amountPaise)
	if err != nil {
		utils.Logf(ctx, "handler: create transaction: error: %v", err)
		switch {
		case errors.Is(err, apperr.ErrValidation):
			utils.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, apperr.ErrNotFound):
			utils.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	resp := dto.TransactionResponse{
		TransactionID:   tx.TransactionID,
		AccountID:       tx.AccountID,
		OperationTypeID: tx.OperationTypeID,
		Amount:          float64(tx.Amount) / 100,
		Type:            tx.Type,
		EventDate:       tx.EventDate.Format("2006-01-02T15:04:05Z07:00"),
	}
	responseBody, err := json.Marshal(resp)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	// Persist the response so future replays return it unchanged.
	if err := h.idempotency.Save(ctx, idempotencyKey, requestHash, http.StatusCreated, responseBody); err != nil {
		utils.Logf(ctx, "handler: idempotency save warning (non-fatal): %v", err)
	}

	utils.Logf(ctx, "handler: create transaction: success transaction_id=%d", tx.TransactionID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(responseBody)
}
