// Package kafka provides the Kafka producer and consumer used by the
// events service, plus a startup wait helper for broker readiness.
package kafka

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// Producer publishes messages to Kafka topics.
type Producer interface {
	Publish(ctx context.Context, topic, key string, payload []byte, headers map[string]string) error
}

// WriterProducer implements Producer on top of kafka-go's Writer.
type WriterProducer struct {
	writer *kafkago.Writer
}

// NewProducer creates a Producer writing to the given brokers.
func NewProducer(brokers []string) *WriterProducer {
	return &WriterProducer{writer: &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		RequiredAcks: kafkago.RequireOne,
	}}
}

// Publish writes a single message with the given key, payload and headers
// to the topic.
func (p *WriterProducer) Publish(ctx context.Context, topic, key string, payload []byte, headers map[string]string) error {
	msg := kafkago.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
	}
	for k, v := range headers {
		msg.Headers = append(msg.Headers, kafkago.Header{Key: k, Value: []byte(v)})
	}
	return p.writer.WriteMessages(ctx, msg)
}

// Close closes the underlying writer.
func (p *WriterProducer) Close() error {
	return p.writer.Close()
}

// MessageFormatter renders a consumed Kafka message for logging. A
// formatting error is logged and the message is still committed, so one
// bad record cannot kill the consumer.
type MessageFormatter func(key, value []byte, headers map[string]string) (string, error)

// messageReader abstracts kafka-go's Reader for tests.
type messageReader interface {
	FetchMessage(ctx context.Context) (kafkago.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafkago.Message) error
	Close() error
}

// Consumer consumes messages from a single topic and logs them.
type Consumer struct {
	reader    messageReader
	topic     string
	formatter MessageFormatter
}

// NewConsumer creates a Consumer reading the topic as part of the given
// consumer group. A nil formatter logs the raw key and value.
func NewConsumer(brokers []string, groupID, topic string, formatter MessageFormatter) *Consumer {
	return newConsumer(kafkago.NewReader(kafkago.ReaderConfig{
		Brokers: brokers,
		GroupID: groupID,
		Topic:   topic,
	}), topic, formatter)
}

func newConsumer(reader messageReader, topic string, formatter MessageFormatter) *Consumer {
	if formatter == nil {
		formatter = func(key, value []byte, _ map[string]string) (string, error) {
			return fmt.Sprintf("key=%s value=%s", key, value), nil
		}
	}
	return &Consumer{reader: reader, topic: topic, formatter: formatter}
}

// Run consumes messages until the context is cancelled, logging each
// consumed message and committing its offset. Formatter errors are logged
// and skipped: the offset is still committed and the loop keeps running.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return fmt.Errorf("fetch from topic %s: %w", c.topic, err)
		}
		headers := make(map[string]string, len(msg.Headers))
		for _, h := range msg.Headers {
			headers[h.Key] = string(h.Value)
		}
		formatted, err := c.formatter(msg.Key, msg.Value, headers)
		if err != nil {
			log.Printf("failed to format message from topic %s (partition %d, offset %d): %v",
				msg.Topic, msg.Partition, msg.Offset, err)
		} else {
			log.Printf("consumed topic=%s partition=%d offset=%d %s",
				msg.Topic, msg.Partition, msg.Offset, formatted)
		}
		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			log.Printf("commit offset for topic %s: %v", c.topic, err)
		}
	}
}

// Close closes the underlying reader.
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// DialFunc establishes a network connection; injectable for tests.
type DialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// WaitForBroker blocks until at least one of the brokers accepts a TCP
// connection, retrying up to attempts times with interval between
// attempts. A nil dial uses a net.Dialer with a per-attempt timeout.
func WaitForBroker(ctx context.Context, brokers []string, attempts int, interval time.Duration, dial DialFunc) error {
	if dial == nil {
		d := &net.Dialer{Timeout: 5 * time.Second}
		dial = d.DialContext
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		for _, broker := range brokers {
			conn, err := dial(ctx, "tcp", broker)
			if err != nil {
				lastErr = err
				continue
			}
			conn.Close()
			return nil
		}
		log.Printf("Kafka brokers %v not ready (attempt %d/%d): %v", brokers, attempt, attempts, lastErr)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
	return fmt.Errorf("kafka brokers %v unreachable after %d attempts: %w", brokers, attempts, lastErr)
}
