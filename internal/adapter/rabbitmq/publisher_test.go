package rabbitmq

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DovCaio/saldo/internal/transaction"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestNewPublisherConfirmError(t *testing.T) {
	_, err := NewPublisher(&mockChannel{confirmErr: errors.New("confirm failed")})
	if err == nil {
		t.Fatal("expected confirm error")
	}
}

func TestNewPublisherSuccess(t *testing.T) {
	publisher, err := NewPublisher(&mockChannel{confirms: make(chan amqp.Confirmation, 1)})
	if err != nil {
		t.Fatalf("NewPublisher() error = %v", err)
	}
	if publisher == nil {
		t.Fatal("expected publisher")
	}
}

func TestPublisherDeclareTopologyErrors(t *testing.T) {
	tests := []struct {
		name string
		ch   *mockChannel
	}{
		{name: "exchange", ch: &mockChannel{exchangeErr: errors.New("exchange")}},
		{name: "dead queue", ch: &mockChannel{queueDeclareErrs: []error{errors.New("dead queue")}}},
		{name: "queue bind", ch: &mockChannel{queueBindErr: errors.New("bind")}},
		{name: "request queue", ch: &mockChannel{queueDeclareErrs: []error{nil, errors.New("request queue")}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := &Publisher{channel: tt.ch}
			if err := publisher.declareTopology(); err == nil {
				t.Fatal("expected topology error")
			}
		})
	}
}

func TestPublishSuccess(t *testing.T) {
	confirms := make(chan amqp.Confirmation, 1)
	confirms <- amqp.Confirmation{Ack: true}
	publisher := &Publisher{
		channel:  &mockChannel{confirms: confirms},
		confirms: confirms,
	}

	err := publisher.Publish(context.Background(), transaction.Transaction{
		ID:          "tx-1",
		Description: "Mercado",
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
}

func TestPublishInvalidTransaction(t *testing.T) {
	publisher := &Publisher{channel: &mockChannel{}}
	err := publisher.Publish(context.Background(), transaction.Transaction{})
	if err == nil {
		t.Fatal("expected invalid transaction error")
	}
}

func TestPublishTopologyError(t *testing.T) {
	publisher := &Publisher{channel: &mockChannel{exchangeErr: errors.New("exchange")}}
	err := publisher.Publish(context.Background(), transaction.Transaction{
		ID:          "tx-1",
		Description: "Mercado",
	})
	if err == nil {
		t.Fatal("expected topology error")
	}
}

func TestPublishBrokerError(t *testing.T) {
	publisher := &Publisher{channel: &mockChannel{publishErr: errors.New("publish")}}
	err := publisher.Publish(context.Background(), transaction.Transaction{
		ID:          "tx-1",
		Description: "Mercado",
	})
	if err == nil {
		t.Fatal("expected publish error")
	}
}

func TestPublishRejected(t *testing.T) {
	confirms := make(chan amqp.Confirmation, 1)
	confirms <- amqp.Confirmation{Ack: false}
	publisher := &Publisher{
		channel:  &mockChannel{confirms: confirms},
		confirms: confirms,
	}

	err := publisher.Publish(context.Background(), transaction.Transaction{
		ID:          "tx-1",
		Description: "Mercado",
	})
	if err == nil {
		t.Fatal("expected rejected confirmation")
	}
}

func TestPublishConfirmationChannelClosed(t *testing.T) {
	confirms := make(chan amqp.Confirmation)
	close(confirms)
	publisher := &Publisher{
		channel:  &mockChannel{confirms: confirms},
		confirms: confirms,
	}

	err := publisher.Publish(context.Background(), transaction.Transaction{
		ID:          "tx-1",
		Description: "Mercado",
	})
	if err == nil {
		t.Fatal("expected closed confirmation channel error")
	}
}

func TestPublishContextCanceled(t *testing.T) {
	confirms := make(chan amqp.Confirmation)
	publisher := &Publisher{
		channel:  &mockChannel{confirms: confirms},
		confirms: confirms,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	err := publisher.Publish(ctx, transaction.Transaction{
		ID:          "tx-1",
		Description: "Mercado",
	})
	if err == nil {
		t.Fatal("expected context error")
	}
}
