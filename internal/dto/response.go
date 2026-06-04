package dto

// ErrorResponse is the shape of all error replies.
type ErrorResponse struct {
	Error string `json:"error" example:"account not found"`
}

// AccountResponse is the response body for account endpoints.
type AccountResponse struct {
	AccountID      int64  `json:"account_id"      example:"1"`
	DocumentNumber string `json:"document_number" example:"12345678900"`
}

// TransactionResponse is the response body for transaction endpoints.
type TransactionResponse struct {
	TransactionID   int64   `json:"transaction_id"    example:"1"`
	AccountID       int64   `json:"account_id"        example:"1"`
	OperationTypeID int64   `json:"operation_type_id" example:"1"`
	Amount          float64 `json:"amount"            example:"112.34"`
	Type            string  `json:"type"              example:"debit"`
	EventDate       string  `json:"event_date"        example:"2026-05-28T17:00:00Z"`
}
