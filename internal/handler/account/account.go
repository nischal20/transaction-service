package account

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nischalpatel/transactions-api/internal/apperr"
	"github.com/nischalpatel/transactions-api/internal/dto"
	svcaccount "github.com/nischalpatel/transactions-api/internal/service/account"
	"github.com/nischalpatel/transactions-api/internal/utils"
)

// AccountHandler holds HTTP handlers for account endpoints.
type AccountHandler struct {
	svc svcaccount.AccountServicer
}

func NewAccountHandler(svc svcaccount.AccountServicer) *AccountHandler {
	return &AccountHandler{svc: svc}
}

// CreateAccount handles POST /accounts
//
//	@Summary		Create an account
//	@Description	Creates a new cardholder account with the given document number
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Param			body	body		dto.CreateAccountRequest	true	"Account payload"
//	@Success		201		{object}	dto.AccountResponse
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		405		{object}	dto.ErrorResponse
//	@Router			/accounts [post]
func (h *AccountHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var body dto.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.Logf(r.Context(), "handler: create account: decode error: %v", err)
		utils.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	utils.Logf(r.Context(), "handler: create account: document_number=%q", body.DocumentNumber)

	ctx := utils.WithActor(r.Context(), r.Header.Get("X-User-ID"))
	acc, err := h.svc.CreateAccount(ctx, body.DocumentNumber)
	if err != nil {
		utils.Logf(r.Context(), "handler: create account: error: %v", err)
		switch {
		case errors.Is(err, apperr.ErrValidation):
			utils.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, apperr.ErrConflict):
			utils.WriteError(w, http.StatusConflict, err.Error())
		default:
			utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	utils.Logf(r.Context(), "handler: create account: success account_id=%d", acc.AccountID)
	utils.WriteJSON(w, http.StatusCreated, dto.AccountResponse{
		AccountID:      acc.AccountID,
		DocumentNumber: acc.DocumentNumber,
	})
}

// GetAccount handles GET /accounts/{accountId}
//
//	@Summary		Get an account
//	@Description	Returns a single account by its numeric ID
//	@Tags			accounts
//	@Produce		json
//	@Param			accountId	path		int	true	"Account ID"
//	@Success		200			{object}	dto.AccountResponse
//	@Failure		400			{object}	dto.ErrorResponse
//	@Failure		404			{object}	dto.ErrorResponse
//	@Failure		405			{object}	dto.ErrorResponse
//	@Router			/accounts/{accountId} [get]
func (h *AccountHandler) GetAccount(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "accountId")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		utils.Logf(r.Context(), "handler: get account: invalid id %q", idStr)
		utils.WriteError(w, http.StatusBadRequest, "invalid account id")
		return
	}
	utils.Logf(r.Context(), "handler: get account: account_id=%d", id)

	acc, err := h.svc.GetAccount(r.Context(), id)
	if err != nil {
		utils.Logf(r.Context(), "handler: get account: error: %v", err)
		switch {
		case errors.Is(err, apperr.ErrValidation):
			utils.WriteError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, apperr.ErrNotFound):
			utils.WriteError(w, http.StatusNotFound, err.Error())
		default:
			utils.WriteError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	utils.Logf(r.Context(), "handler: get account: success document_number=%q", acc.DocumentNumber)
	utils.WriteJSON(w, http.StatusOK, dto.AccountResponse{
		AccountID:      acc.AccountID,
		DocumentNumber: acc.DocumentNumber,
	})
}
