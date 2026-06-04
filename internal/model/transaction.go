package model

import "time"

// OperationType IDs as defined in the requirements.
const (
	OperationNormalPurchase       = 1
	OperationPurchaseInstallments = 2
	OperationWithdrawal           = 3
	OperationCreditVoucher        = 4
)

// OperationType describes the kind of transaction.
type OperationType struct {
	OperationTypeID int64  `json:"operation_type_id"`
	Description     string `json:"description"`
}

// Transaction represents a financial operation linked to an account.
type Transaction struct {
	TransactionID   int64     `json:"transaction_id"`
	AccountID       int64     `json:"account_id"`
	OperationTypeID int64     `json:"operation_type_id"`
	Amount          int64     `json:"amount"`
	Type            string    `json:"type"`
	EventDate       time.Time `json:"event_date"`
	CreatedAt       time.Time `json:"created_at"`
}

// TransactionType returns "debit" or "credit" for the given operation type.
func TransactionType(operationTypeID int64) string {
	if IsDebit(operationTypeID) {
		return "debit"
	}
	return "credit"
}

// SeedOperationTypes returns the pre-defined operation types from the spec.
func SeedOperationTypes() []OperationType {
	return []OperationType{
		{OperationTypeID: OperationNormalPurchase, Description: "Normal Purchase"},
		{OperationTypeID: OperationPurchaseInstallments, Description: "Purchase with installments"},
		{OperationTypeID: OperationWithdrawal, Description: "Withdrawal"},
		{OperationTypeID: OperationCreditVoucher, Description: "Credit Voucher"},
	}
}

// IsDebit returns true for operation types that should be stored as negative amounts.
func IsDebit(operationTypeID int64) bool {
	switch operationTypeID {
	case OperationNormalPurchase, OperationPurchaseInstallments, OperationWithdrawal:
		return true
	}
	return false
}
