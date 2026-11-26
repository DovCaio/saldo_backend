package application

import (
	"errors"
)

var (
	UninitializedPublisher = errors.New("the publisher wasnt initialized")
)
