package model

import "time"

// Account represents a cardholder account.
type Account struct {
	AccountID      int64     `json:"account_id"`
	DocumentNumber string    `json:"document_number"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
