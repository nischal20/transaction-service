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
	svctransaction "github.com/nischalpatel/transactions-api/internal/service/transaction"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// TransactionHandler holds HTTP handlers for transaction endpoints.
type TransactionHandler struct {
	svc svctransaction.TransactionServicer
}

func NewTransactionHandler(svc svctransaction.TransactionServicer) *TransactionHandler {
	return &TransactionHandler{svc: svc}
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
//	@Failure		409					{object}	dto.ErrorResponse
//	@Failure		422					{object}	dto.ErrorResponse
//	@Router			/transactions [post]
func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	idemKey := r.Header.Get("X-Idempotency-Key")
	if idemKey == "" {
		utils.WriteError(w, http.StatusBadRequest, "X-Idempotency-Key header is required")
		return
	}

	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	sum := sha256.Sum256(rawBody)
	idemHash := hex.EncodeToString(sum[:])

	var body dto.CreateTransactionRequest
	if err := json.NewDecoder(bytes.NewReader(rawBody)).Decode(&body); err != nil {
		utils.Logf(r.Context(), "handler: create transaction: decode error: %v", err)
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	amountPaise := int64(math.Round(body.Amount * 100))
	ctx := utils.WithActor(r.Context(), r.Header.Get("X-User-ID"))

	tx, err := h.svc.CreateTransaction(ctx, body.AccountID, body.OperationTypeID, amountPaise, idemKey, idemHash)
	if err != nil {
		switch {
		case errors.Is(err, apperr.ErrValidation):
			utils.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, apperr.ErrConflict):
			utils.WriteError(w, http.StatusConflict, err.Error())
		case errors.Is(err, apperr.ErrNotFound):
			utils.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	utils.WriteJSON(w, http.StatusCreated, dto.TransactionResponse{
		TransactionID:   tx.TransactionID,
		AccountID:       tx.AccountID,
		OperationTypeID: tx.OperationTypeID,
		Amount:          float64(tx.Amount) / 100,
		Type:            tx.Type,
		EventDate:       tx.EventDate.Format("2006-01-02T15:04:05Z07:00"),
	})
}
