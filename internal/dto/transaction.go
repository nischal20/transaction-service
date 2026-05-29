package dto

// CreateTransactionRequest is the request body for POST /transactions.
type CreateTransactionRequest struct {
	AccountID       int64   `json:"account_id"`
	OperationTypeID int64   `json:"operation_type_id"`
	Amount          float64 `json:"amount"`
}
