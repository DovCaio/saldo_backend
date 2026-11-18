package application

import (
	"context"

	"github.com/DovCaio/saldo/internal/transaction"
)


type MessagingPublisher  interface {

	Publish(ctx context.Context, transaction transaction.Transaction) error

}