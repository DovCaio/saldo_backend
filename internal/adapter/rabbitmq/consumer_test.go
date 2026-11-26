package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ackMock struct {
	ackCalls  int
	nackCalls int
	ackErr    error
	nackErr   error
}

func (a *ackMock) Ack(tag uint64, multiple bool) error {
	a.ackCalls++
	return a.ackErr
}

func (a *ackMock) Nack(tag uint64, multiple bool, requeue bool) error {
	a.nackCalls++
	return a.nackErr
}

func (a *ackMock) Reject(tag uint64, requeue bool) error {
	return nil
}

func validResult() ClassificationResult {
	return ClassificationResult{
		TransactionID:      "tx-1",
		EventID:            "evt-1",
		Category:           "food",
		Confidence:         0.8,
		DescriptionVersion: 1,
	}
}

func TestNewConsumerValidation(t *testing.T) {
	handler := func(ctx context.Context, result ClassificationResult) error {
		return nil
	}

	if _, err := NewConsumer(nil, handler); err == nil {
		t.Error("expected error for nil channel")
	}

	if _, err := NewConsumer(&mockChannel{}, nil); err == nil {
		t.Error("expected error for nil handler")
	}
}

func TestNewConsumerDeclaresTopology(t *testing.T) {
	consumer, err := NewConsumer(&mockChannel{}, func(ctx context.Context, result ClassificationResult) error {
		return nil
	})
	if err != nil {
		t.Fatalf("NewConsumer() error = %v", err)
	}
	if consumer == nil {
		t.Fatal("expected consumer")
	}
}

func TestConsumerDeclareTopologyErrors(t *testing.T) {
	tests := []struct {
		name string
		ch   *mockChannel
	}{
		{name: "exchange", ch: &mockChannel{exchangeErr: errors.New("exchange")}},
		{name: "dead queue", ch: &mockChannel{queueDeclareErrs: []error{errors.New("dead queue")}}},
		{name: "queue bind", ch: &mockChannel{queueBindErr: errors.New("bind")}},
		{name: "result queue", ch: &mockChannel{queueDeclareErrs: []error{nil, errors.New("result queue")}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewConsumer(tt.ch, func(ctx context.Context, result ClassificationResult) error {
				return nil
			})
			if err == nil {
				t.Fatal("expected topology error")
			}
		})
	}
}

func TestConsumeQosError(t *testing.T) {
	consumer := &Consumer{channel: &mockChannel{qosErr: errors.New("qos")}}
	if err := consumer.Consume(context.Background()); err == nil {
		t.Fatal("expected qos error")
	}
}

func TestConsumeStartError(t *testing.T) {
	consumer := &Consumer{channel: &mockChannel{consumeErr: errors.New("consume")}}
	if err := consumer.Consume(context.Background()); err == nil {
		t.Fatal("expected consume error")
	}
}

func TestConsumeStopsWhenContextCanceled(t *testing.T) {
	ch := &mockChannel{deliveries: make(chan amqp.Delivery)}
	consumer := &Consumer{
		channel: ch,
		handler: func(ctx context.Context, result ClassificationResult) error {
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- consumer.Consume(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Consume() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Consume() did not return after cancel")
	}
}

func TestConsumeHandlesDeliveryThenClosedChannel(t *testing.T) {
	deliveries := make(chan amqp.Delivery)
	ack := &ackMock{}
	body, err := json.Marshal(validResult())
	if err != nil {
		t.Fatal(err)
	}

	consumer := &Consumer{
		channel: &mockChannel{deliveries: deliveries},
		handler: func(ctx context.Context, result ClassificationResult) error {
			return nil
		},
	}

	done := make(chan error, 1)
	go func() {
		done <- consumer.Consume(context.Background())
	}()

	deliveries <- amqp.Delivery{Acknowledger: ack, Body: body}
	close(deliveries)

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected closed delivery channel error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Consume() did not return")
	}

	if ack.ackCalls != 1 {
		t.Errorf("ack calls = %d, want 1", ack.ackCalls)
	}
}

func TestHandleDeliveryAcksValidResult(t *testing.T) {
	ack := &ackMock{}
	var got ClassificationResult
	consumer := &Consumer{
		handler: func(ctx context.Context, result ClassificationResult) error {
			got = result
			return nil
		},
	}

	body, err := json.Marshal(validResult())
	if err != nil {
		t.Fatal(err)
	}

	consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		Body:         body,
	})

	if ack.ackCalls != 1 {
		t.Errorf("ack calls = %d, want 1", ack.ackCalls)
	}
	if ack.nackCalls != 0 {
		t.Errorf("nack calls = %d, want 0", ack.nackCalls)
	}
	if got.TransactionID != "tx-1" {
		t.Errorf("handled transaction = %q", got.TransactionID)
	}
}

func TestHandleDeliveryNacksInvalidJSON(t *testing.T) {
	ack := &ackMock{}
	consumer := &Consumer{
		handler: func(ctx context.Context, result ClassificationResult) error {
			t.Fatal("handler should not be called")
			return nil
		},
	}

	consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		Body:         []byte("{"),
	})

	if ack.nackCalls != 1 {
		t.Errorf("nack calls = %d, want 1", ack.nackCalls)
	}
}

func TestHandleDeliveryLogsNackErrorOnInvalidJSON(t *testing.T) {
	ack := &ackMock{nackErr: errors.New("nack failed")}
	consumer := &Consumer{handler: func(ctx context.Context, result ClassificationResult) error { return nil }}

	consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		Body:         []byte("{"),
	})

	if ack.nackCalls != 1 {
		t.Errorf("nack calls = %d, want 1", ack.nackCalls)
	}
}

func TestHandleDeliveryLogsNackErrorOnInvalidPayload(t *testing.T) {
	ack := &ackMock{nackErr: errors.New("nack failed")}
	consumer := &Consumer{handler: func(ctx context.Context, result ClassificationResult) error { return nil }}
	body, err := json.Marshal(ClassificationResult{EventID: "e", Category: "c", Confidence: 0.5, DescriptionVersion: 1})
	if err != nil {
		t.Fatal(err)
	}

	consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		Body:         body,
	})

	if ack.nackCalls != 1 {
		t.Errorf("nack calls = %d, want 1", ack.nackCalls)
	}
}

func TestHandleDeliveryNacksInvalidPayload(t *testing.T) {
	tests := []struct {
		name   string
		result ClassificationResult
	}{
		{name: "missing transaction id", result: ClassificationResult{EventID: "e", Category: "c", Confidence: 0.5, DescriptionVersion: 1}},
		{name: "missing event id", result: ClassificationResult{TransactionID: "t", Category: "c", Confidence: 0.5, DescriptionVersion: 1}},
		{name: "missing category", result: ClassificationResult{TransactionID: "t", EventID: "e", Confidence: 0.5, DescriptionVersion: 1}},
		{name: "confidence too low", result: ClassificationResult{TransactionID: "t", EventID: "e", Category: "c", Confidence: -0.1, DescriptionVersion: 1}},
		{name: "confidence too high", result: ClassificationResult{TransactionID: "t", EventID: "e", Category: "c", Confidence: 1.1, DescriptionVersion: 1}},
		{name: "invalid description version", result: ClassificationResult{TransactionID: "t", EventID: "e", Category: "c", Confidence: 0.5, DescriptionVersion: 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ack := &ackMock{}
			consumer := &Consumer{
				handler: func(ctx context.Context, result ClassificationResult) error {
					t.Fatal("handler should not be called")
					return nil
				},
			}

			body, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatal(err)
			}

			consumer.handleDelivery(context.Background(), amqp.Delivery{
				Acknowledger: ack,
				Body:         body,
			})

			if ack.nackCalls != 1 {
				t.Errorf("nack calls = %d, want 1", ack.nackCalls)
			}
		})
	}
}

func TestHandleDeliveryNacksHandlerError(t *testing.T) {
	ack := &ackMock{}
	consumer := &Consumer{
		handler: func(ctx context.Context, result ClassificationResult) error {
			return errors.New("update failed")
		},
	}

	body, err := json.Marshal(validResult())
	if err != nil {
		t.Fatal(err)
	}

	consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		Body:         body,
	})

	if ack.nackCalls != 1 {
		t.Errorf("nack calls = %d, want 1", ack.nackCalls)
	}
}

func TestHandleDeliveryLogsAckError(t *testing.T) {
	ack := &ackMock{ackErr: errors.New("ack failed")}
	consumer := &Consumer{
		handler: func(ctx context.Context, result ClassificationResult) error {
			return nil
		},
	}

	body, err := json.Marshal(validResult())
	if err != nil {
		t.Fatal(err)
	}

	consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		Body:         body,
	})

	if ack.ackCalls != 1 {
		t.Errorf("ack calls = %d, want 1", ack.ackCalls)
	}
}

func TestHandleDeliveryLogsNackError(t *testing.T) {
	ack := &ackMock{nackErr: errors.New("nack failed")}
	consumer := &Consumer{
		handler: func(ctx context.Context, result ClassificationResult) error {
			return errors.New("handler failed")
		},
	}

	body, err := json.Marshal(validResult())
	if err != nil {
		t.Fatal(err)
	}

	consumer.handleDelivery(context.Background(), amqp.Delivery{
		Acknowledger: ack,
		Body:         body,
	})

	if ack.nackCalls != 1 {
		t.Errorf("nack calls = %d, want 1", ack.nackCalls)
	}
}

func TestConnectionCloseWithNilFields(t *testing.T) {
	conn := &Connection{}
	if err := conn.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if conn.Channel() != nil {
		t.Error("expected nil channel")
	}
}

func TestNewConnectionInvalidURL(t *testing.T) {
	got, err := NewConnection("amqp://invalid-host:1/")
	if !errors.Is(err, RabbitMQConnectionError) {
		t.Fatalf("error = %v, want %v", err, RabbitMQConnectionError)
	}
	if got != nil {
		t.Errorf("connection = %#v, want nil", got)
	}
}
