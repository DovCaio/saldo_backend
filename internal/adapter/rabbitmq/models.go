package rabbitmq

import (
	"fmt"

	"github.com/DovCaio/saldo/internal/transaction"
	"github.com/google/uuid"
)

type ClassificationRequest struct {
	TransactionID     string `json:"transaction_id"`
	EventID           string `json:"event_id"`
	Description       string `json:"description"`
	DescriptionVersion int64 `json:"description_version"`
}

func FromTransaction(transaction transaction.Transaction) (*ClassificationRequest, error) {
	if transaction.ID == "" ||
		transaction.Description == "" {
		return nil, fmt.Errorf("invalid transaction")
	}

	return &ClassificationRequest{
		TransactionID: transaction.ID,
		EventID: uuid.New().String(),
		Description: transaction.Description,
		DescriptionVersion: 1,
	}, nil
}


type ClassificationResult struct {
    TransactionID      string  `json:"transaction_id"`
    EventID           string  `json:"event_id"`
    Category           string  `json:"category"`
    Confidence         float64 `json:"confidence"`
    DescriptionVersion int64   `json:"description_version"`
}