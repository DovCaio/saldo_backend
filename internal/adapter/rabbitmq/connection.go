package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Connection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewConnection(url string) (*Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		fmt.Printf("rabbit connection error: %v", err)
		return nil, RabbitMQConnectionError
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		fmt.Printf("rabbit channel error: %v", err)
	}

	return &Connection{
		conn:    conn,
		channel: channel,
	}, nil
}

func (c *Connection) Channel() *amqp.Channel {
	return c.channel
}

func (c *Connection) Close() error {

	if c.channel != nil {
		if err := c.channel.Close(); err != nil {

			c.conn.Close()
			return fmt.Errorf("clossing RabbitMQ connection erro: %v", err)

		}
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("closing RabbitMQ connection: %w", err)
		}
	}

	return nil

}
