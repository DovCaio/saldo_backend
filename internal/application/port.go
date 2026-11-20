package application

import (
	"context"

	"github.com/DovCaio/saldo/internal/transaction"
)


type MessagingPublisher  interface {

	Publish(ctx context.Context, transaction transaction.Transaction) error

}

type ClassificationRepository interface {
    UpdateClassification(
        ctx context.Context,
        transactionID string,
        category string,
        confidence float64,
        descriptionVersion int64,
    ) error
}