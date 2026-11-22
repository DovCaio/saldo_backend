package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
    resultQueue          = "transactions.classification.completed"
    resultDeadQueue     = "transactions.classification.completed.dead"
    resultDeadExchange  = "transactions.classification.completed"
    resultDeadRoutingKey = "classification.result.dead"
)

type ClassificationHandler func(
    ctx context.Context,
    result ClassificationResult,
) error

type Consumer struct {
    channel  *amqp.Channel
    handler ClassificationHandler
}

func NewConsumer(
    channel *amqp.Channel,
    handler ClassificationHandler,
) (*Consumer, error) {
    if channel == nil {
        return nil, fmt.Errorf("rabbitmq channel is nil")
    }

    if handler == nil {
        return nil, fmt.Errorf("classification handler is nil")
    }

    consumer := &Consumer{
        channel:  channel,
        handler: handler,
    }

    if err := consumer.declareTopology(); err != nil {
        return nil, err
    }

    return consumer, nil
}

func (c *Consumer) declareTopology() error {
    if err := c.channel.ExchangeDeclare(
        resultDeadExchange,
        "direct",
        true,
        false,
        false,
        false,
        nil,
    ); err != nil {
        return fmt.Errorf(
            "declaring classification result dead exchange: %w",
            err,
        )
    }

    _, err := c.channel.QueueDeclare(
        resultDeadQueue,
        true,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        return fmt.Errorf(
            "declaring classification result dead queue: %w",
            err,
        )
    }

    if err := c.channel.QueueBind(
        resultDeadQueue,
        resultDeadRoutingKey,
        resultDeadExchange,
        false,
        nil,
    ); err != nil {
        return fmt.Errorf(
            "binding classification result dead queue: %w",
            err,
        )
    }

    _, err = c.channel.QueueDeclare(
        resultQueue,
        true,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        return fmt.Errorf(
            "declaring classification result queue: %w",
            err,
        )
    }

    return nil
}

func (c *Consumer) Consume(ctx context.Context) error {
    if err := c.channel.Qos(1, 0, false); err != nil {
        return fmt.Errorf("setting consumer QoS: %w", err)
    }

    deliveries, err := c.channel.ConsumeWithContext(
        ctx,
        resultQueue,
        "saldo-classification-consumer",
        false, // autoAck
        false, // exclusive
        false, // noLocal
        false, // noWait
        nil,   // args
    )
    if err != nil {
        return fmt.Errorf("starting classification consumer: %w", err)
    }

    log.Println("Listening for classification results...")

    for {
        select {
        case <-ctx.Done():
            log.Println("Classification consumer shutting down...")
            return nil

        case delivery, ok := <-deliveries:
            if !ok {
                return fmt.Errorf("classification delivery channel closed")
            }

            c.handleDelivery(ctx, delivery)
        }
    }
}

func (c *Consumer) handleDelivery(
    ctx context.Context,
    delivery amqp.Delivery,
) {
    var result ClassificationResult

    if err := json.Unmarshal(delivery.Body, &result); err != nil {
        log.Printf("Invalid classification result: %v", err)

        if nackErr := delivery.Nack(false, false); nackErr != nil {
            log.Printf("Failed to nack invalid message: %v", nackErr)
        }

        return
    }

    if err := validateResult(result); err != nil {
        log.Printf("Invalid classification payload: %v", err)

        if nackErr := delivery.Nack(false, false); nackErr != nil {
            log.Printf("Failed to nack invalid payload: %v", nackErr)
        }

        return
    }

    if err := c.handler(ctx, result); err != nil {
        log.Printf(
            "Failed to process classification for transaction %s: %v",
            result.TransactionID,
            err,
        )

        if nackErr := delivery.Nack(false, false); nackErr != nil {
            log.Printf("Failed to nack classification result: %v", nackErr)
        }

        return
    }

    if err := delivery.Ack(false); err != nil {
        log.Printf(
            "Failed to ack classification result %s: %v",
            result.EventID,
            err,
        )

        return
    }

    log.Printf(
        "Classification completed for transaction %s: %s",
        result.TransactionID,
        result.Category,
    )
}

func validateResult(result ClassificationResult) error {
    if result.TransactionID == "" {
        return fmt.Errorf("transaction_id is required")
    }

    if result.EventID == "" {
        return fmt.Errorf("event_id is required")
    }

    if result.Category == "" {
        return fmt.Errorf("category is required")
    }

    if result.Confidence < 0 || result.Confidence > 1 {
        return fmt.Errorf("confidence must be between 0 and 1")
    }

    if result.DescriptionVersion < 1 {
        return fmt.Errorf("description_version must be greater than 0")
    }

    return nil
}