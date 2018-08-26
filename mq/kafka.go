package mq

import (
	"errors"
)

// errors
var (
	ErrNoProducer = errors.New("no producer")
	ErrNoConsumer = errors.New("no consumer")
)

// Messages receive messages
func (mq *MessageQueue) Messages() <-chan string {
	return mq.messages
}

// Close message queue
func (mq *MessageQueue) Close() {
	if mq.c != nil {
		mq.c.Close()
	}
	if mq.p != nil {
		mq.p.Close()
	}
}
