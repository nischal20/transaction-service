package dto

// CreateAccountRequest is the request body for POST /accounts.
type CreateAccountRequest struct {
	DocumentNumber string `json:"document_number"`
}
