// +build test

package mq

import (
	"errors"
	"os"
)

type dummyCloseable struct {
}

// Close dummy
func (*dummyCloseable) Close() {
}

// MessageQueue is the message queue sturcture (dummy)
type MessageQueue struct {
	messages chan string
	p *dummyCloseable
	c *dummyCloseable
}

var ErrDummyMQ = errors.New("dummy MQ")

func NewProducer(servers []string, topic string) (*MessageQueue, error) {
	return nil, ErrDummyMQ
}

// NewConsumer init consumer
func NewConsumer(servers []string, topic, groupID string, sigchan chan os.Signal) (*MessageQueue, error) {
	return nil, ErrDummyMQ
}

// Publish message
func (mq *MessageQueue) Publish(message string) error {
	return ErrDummyMQ
}
