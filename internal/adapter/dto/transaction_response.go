package dto

import (
	"time"
)

type TransactionTypeResponse string

const (
	Income  TransactionTypeResponse = "INCOME"
	Expense TransactionTypeResponse = "EXPENSE"
)

type TransactionResponse struct {
	ID          string                  `json:"id"`
	Date        time.Time               `json:"date"`
	Description string                  `json:"description"`
	AmountCents int64                   `json:"amount_cents"`
	Type        TransactionTypeResponse `json:"type"`
	Category    string                  `json:"category"`
}

func NewTransactionResponse(id string, date time.Time, description string, amountCents int64, type_ string, category string) (*TransactionResponse, error) {

	var responseType TransactionTypeResponse

	switch type_ {
	case "INCOME", "EXPENSE":
		responseType = TransactionTypeResponse(type_)
	default:
		return nil, InvalidResponse
	}

	return &TransactionResponse{
		ID:          id,
		Date:        date,
		Description: description,
		AmountCents: amountCents,
		Type:        responseType,
		Category:    category,
	}, nil

}
