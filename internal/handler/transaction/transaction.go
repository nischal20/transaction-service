package transaction

import (
	"encoding/json"
	"errors"
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
//	@Description	Send a positive amount. The server applies the correct sign based on operation type.
//	@Tags			transactions
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.CreateTransactionRequest	true	"Transaction payload"
//	@Success		201		{object}	dto.TransactionResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		405		{object}	dto.ErrorResponse
//	@Failure		422		{object}	dto.ErrorResponse
//	@Router			/transactions [post]
func (h *TransactionHandler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.Logf(r.Context(), "handler: create transaction: decode error: %v", err)
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	utils.Logf(r.Context(), "handler: create transaction: account_id=%d op_type=%d amount=%.2f", body.AccountID, body.OperationTypeID, body.Amount)

	tx, err := h.svc.CreateTransaction(r.Context(), body.AccountID, body.OperationTypeID, body.Amount)
	if err != nil {
		utils.Logf(r.Context(), "handler: create transaction: error: %v", err)
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

	utils.Logf(r.Context(), "handler: create transaction: success transaction_id=%d", tx.TransactionID)
	utils.WriteJSON(w, http.StatusCreated, tx)
}
