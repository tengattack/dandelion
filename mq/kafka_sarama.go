// +build !cgo,!test windows,!test

package mq

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tengattack/dandelion/log"

	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
)

// MessageQueue is the message queue sturcture
type MessageQueue struct {
	Topic    string
	Servers  []string
	messages chan string
	p        sarama.AsyncProducer
	c        *cluster.Consumer
}

func getPartitions(c sarama.Consumer, topic, partitions string) ([]int32, error) {
	if partitions == "all" {
		return c.Partitions(topic)
	}

	tmp := strings.Split(partitions, ",")
	var pList []int32
	for i := range tmp {
		val, err := strconv.ParseInt(tmp[i], 10, 32)
		if err != nil {
			return nil, err
		}
		pList = append(pList, int32(val))
	}

	return pList, nil
}

// NewProducer init producer
func NewProducer(servers []string, topic string) (*MessageQueue, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal       // Only wait for the leader to ack
	config.Producer.Compression = sarama.CompressionSnappy   // Compress messages
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms

	producer, err := sarama.NewAsyncProducer(servers, config)
	if err != nil {
		log.LogError.Errorf("Failed to start Sarama producer: %v", err)
		return nil, err
	}

	// We will just log to STDOUT if we're not able to produce messages.
	// Note: messages will only be returned here after all retry attempts are exhausted.
	go func() {
		for err := range producer.Errors() {
			log.LogError.Errorf("Failed to write message: %v", err)
		}
	}()

	mq := MessageQueue{
		Topic:   topic,
		Servers: servers,
		p:       producer,
	}
	return &mq, nil
}

// NewConsumer init consumer
func NewConsumer(servers []string, topic, groupID string, sigchan chan os.Signal) (*MessageQueue, error) {
	// init (custom) config, enable errors and notifications
	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true

	c, err := cluster.NewConsumer(servers, groupID, []string{topic}, config)
	if err != nil {
		return nil, err
	}

	messages := make(chan string)

	// consume errors
	go func() {
		for err := range c.Errors() {
			log.LogError.Errorf("consume error: %v", err)
		}
	}()

	// consume notifications
	go func() {
		for ntf := range c.Notifications() {
			log.LogAccess.Debugf("rebalanced: %+v", ntf)
		}
	}()

	go func() {
		defer close(messages)
		// consume messages, watch signals
		for {
			select {
			case msg, ok := <-c.Messages():
				if ok {
					messages <- string(msg.Value)
					c.MarkOffset(msg, "") // mark message as processed
				}
			case sig := <-sigchan:
				log.LogAccess.Infof("Caught signal %v: terminating", sig)
				return
			}
		}
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
	mq.p.Input() <- &sarama.ProducerMessage{
		Topic: mq.Topic,
		// Key:   sarama.StringEncoder(r.RemoteAddr),
		Value: sarama.StringEncoder(message),
	}
	return nil
}
