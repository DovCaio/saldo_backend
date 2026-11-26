package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type mockChannel struct {
	confirmErr        error
	exchangeErr       error
	queueBindErr      error
	publishErr        error
	qosErr            error
	consumeErr        error
	queueDeclareErrs  []error
	queueDeclareCalls int
	confirms          chan amqp.Confirmation
	deliveries        chan amqp.Delivery
	published         int
}

func (m *mockChannel) Confirm(noWait bool) error {
	return m.confirmErr
}

func (m *mockChannel) NotifyPublish(confirm chan amqp.Confirmation) chan amqp.Confirmation {
	if m.confirms != nil {
		return m.confirms
	}
	return confirm
}

func (m *mockChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	return m.exchangeErr
}

func (m *mockChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	idx := m.queueDeclareCalls
	m.queueDeclareCalls++
	if idx < len(m.queueDeclareErrs) && m.queueDeclareErrs[idx] != nil {
		return amqp.Queue{}, m.queueDeclareErrs[idx]
	}
	return amqp.Queue{Name: name}, nil
}

func (m *mockChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	return m.queueBindErr
}

func (m *mockChannel) PublishWithContext(ctx context.Context, exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	m.published++
	return m.publishErr
}

func (m *mockChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	return m.qosErr
}

func (m *mockChannel) ConsumeWithContext(ctx context.Context, queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	if m.consumeErr != nil {
		return nil, m.consumeErr
	}
	if m.deliveries == nil {
		m.deliveries = make(chan amqp.Delivery)
	}
	return m.deliveries, nil
}
