package rabbitmq

import (
	"errors"
)



var (
	RabbitMQConnectionError = errors.New("connection with rabbit is not working")
	RabbitMQChannelError = errors.New("channel is not opening")
)
