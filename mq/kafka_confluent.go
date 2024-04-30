//go:build cgo && !windows && !test
// +build cgo,!windows,!test

package mq

import (
	"os"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"

	"github.com/tengattack/tgo/logger"
)

// MessageQueue is the message queue sturcture
type MessageQueue struct {
	Topic    string
	Servers  []string
	messages chan string
	p        *kafka.Producer
	c        *kafka.Consumer
}

// NewProducer init producer
func NewProducer(servers []string, topic string) (*MessageQueue, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": strings.Join(servers, ",")})
	if err != nil {
		return nil, err
	}
	// Delivery report handler for produced messages
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					logger.Errorf("Delivery failed: %v", ev.TopicPartition)
				} else {
					logger.Debugf("Delivered message to %v", ev.TopicPartition)
				}
			}
		}
	}()

	mq := MessageQueue{
		Topic:   topic,
		Servers: servers,
		p:       p,
	}
	return &mq, nil
}

// NewConsumer init consumer
func NewConsumer(servers []string, topic, groupID string, sigchan chan os.Signal) (*MessageQueue, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":               strings.Join(servers, ","),
		"group.id":                        groupID,
		"session.timeout.ms":              6000,
		"go.events.channel.enable":        true,
		"go.application.rebalance.enable": true,
		"default.topic.config":            kafka.ConfigMap{"auto.offset.reset": "latest"},
	})

	if err != nil {
		return nil, err
	}

	c.SubscribeTopics([]string{topic}, nil)
	messages := make(chan string)

	go func() {
		run := true
		for run == true {
			select {
			case sig := <-sigchan:
				logger.Infof("Caught signal %v: terminating", sig)
				run = false
			case ev := <-c.Events():
				switch e := ev.(type) {
				case kafka.AssignedPartitions:
					logger.Debug(e)
					c.Assign(e.Partitions)
				case kafka.RevokedPartitions:
					logger.Debug(e)
					c.Unassign()
				case *kafka.Message:
					logger.Debug(e)
					messages <- string(e.Value)
				case kafka.PartitionEOF:
					logger.Debug(e)
				case kafka.Error:
					logger.Error(e)
					if e.Code() == kafka.ErrAllBrokersDown {
						// REVIEW: it will reconnect automatically?
						// run = false
					}
				}
			}
		}
		close(messages)
	}()
	mq := MessageQueue{
		Topic:    topic,
		Servers:  servers,
		c:        c,
		messages: messages,
	}
	return &mq, nil
}

// Publish message
func (mq *MessageQueue) Publish(message string) error {
	if mq.p == nil {
		return ErrNoProducer
	}
	// we do not wait for it delivered
	return mq.p.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &mq.Topic, Partition: kafka.PartitionAny},
		Value:          []byte(message),
		// Headers:        []kafka.Header{{"myTestHeader", []byte("header values are binary")}},
	}, nil)
}
