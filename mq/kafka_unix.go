// +build !windows

package mq

import (
	"os"
	"strings"

	"../log"
	"github.com/confluentinc/confluent-kafka-go/kafka"
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
					log.LogError.Errorf("Delivery failed: %v", ev.TopicPartition)
				} else {
					log.LogAccess.Debugf("Delivered message to %v", ev.TopicPartition)
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
				log.LogAccess.Infof("Caught signal %v: terminating", sig)
				run = false
			case ev := <-c.Events():
				switch e := ev.(type) {
				case kafka.AssignedPartitions:
					log.LogAccess.Debug(e)
					c.Assign(e.Partitions)
				case kafka.RevokedPartitions:
					log.LogAccess.Debug(e)
					c.Unassign()
				case *kafka.Message:
					log.LogAccess.Debug(e)
					messages <- string(e.Value)
				case kafka.PartitionEOF:
					log.LogAccess.Debug(e)
				case kafka.Error:
					log.LogError.Error(e)
					run = false
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
