package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	requestQueue = "transactions.classification.requested"
	deadQueue    = "transactions.classification.dead"
	deadExchange = "transactions.dlx"
	deadRouting  = "classification.dead"
)

type ClassificationRequest struct {
	TransactionID     string `json:"transaction_id"`
	EventID           string `json:"event_id"`
	Description       string `json:"description"`
	DescriptionVersion int64 `json:"description_version"`
}

type Publisher struct {
	channel  *amqp.Channel
	confirms <-chan amqp.Confirmation
	mu       sync.Mutex
}

func NewPublisher(channel *amqp.Channel) (*Publisher, error) {
	if err := channel.Confirm(false); err != nil {
		return nil, fmt.Errorf("enabling publisher confirms: %w", err)
	}

	confirms := channel.NotifyPublish(
		make(chan amqp.Confirmation, 1),
	)

	return &Publisher{
		channel:  channel,
		confirms: confirms,
	}, nil
}

func (p *Publisher) declareTopology() error {
	if err := p.channel.ExchangeDeclare(
		deadExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declaring dead letter exchange: %w", err)
	}

	if _, err := p.channel.QueueDeclare(
		deadQueue,
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("declaring dead letter queue: %w", err)
	}

	if err := p.channel.QueueBind(
		deadQueue,
		deadRouting,
		deadExchange,
		false,
		nil,
	); err != nil {
		return fmt.Errorf("binding dead letter queue: %w", err)
	}

	_, err := p.channel.QueueDeclare(
		requestQueue,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    deadExchange,
			"x-dead-letter-routing-key": deadRouting,
		},
	)
	if err != nil {
		return fmt.Errorf("declaring classification request queue: %w", err)
	}

	return nil
}

func (p *Publisher) Publish(
	ctx context.Context,
	request ClassificationRequest,
) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if request.TransactionID == "" ||
		request.EventID == "" ||
		request.Description == "" ||
		request.DescriptionVersion < 1 {
		return fmt.Errorf("invalid classification request")
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("encoding classification request: %w", err)
	}

	if err := p.declareTopology(); err != nil {
		return err
	}

	err = p.channel.PublishWithContext(
		ctx,
		"",
		requestQueue,
		true,
		false,
		amqp.Publishing{
			ContentType:     "application/json",
			ContentEncoding: "utf-8",
			DeliveryMode:    amqp.Persistent,
			Timestamp:       time.Now(),
			MessageId:       request.EventID,
			Body:            body,
		},
	)
	if err != nil {
		return fmt.Errorf("publishing classification request: %w", err)
	}

	select {
	case confirmation, ok := <-p.confirms:
		if !ok {
			return fmt.Errorf("publisher confirmation channel closed")
		}
		if !confirmation.Ack {
			return fmt.Errorf("RabbitMQ rejected classification request")
		}
		return nil

	case <-ctx.Done():
		return fmt.Errorf(
			"waiting for RabbitMQ confirmation: %w",
			ctx.Err(),
		)
	}
}
