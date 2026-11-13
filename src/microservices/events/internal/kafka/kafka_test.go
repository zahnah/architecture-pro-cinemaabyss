package kafka

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

func TestWaitForBrokerSucceedsAfterFailures(t *testing.T) {
	failures := 3
	calls := 0
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		calls++
		if calls <= failures {
			return nil, errors.New("connection refused")
		}
		c1, c2 := net.Pipe()
		defer c2.Close()
		return c1, nil
	}

	err := WaitForBroker(context.Background(), []string{"kafka:9092"}, 5, time.Millisecond, dial)
	if err != nil {
		t.Fatalf("WaitForBroker returned error: %v", err)
	}
	if calls != failures+1 {
		t.Errorf("dial called %d times, want %d", calls, failures+1)
	}
}

func TestWaitForBrokerExhaustsAttempts(t *testing.T) {
	calls := 0
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		calls++
		return nil, errors.New("connection refused")
	}

	err := WaitForBroker(context.Background(), []string{"kafka:9092"}, 3, time.Millisecond, dial)
	if err == nil {
		t.Fatal("WaitForBroker returned nil error, want exhaustion error")
	}
	if calls != 3 {
		t.Errorf("dial called %d times, want 3", calls)
	}
}

func TestWaitForBrokerTriesEveryBroker(t *testing.T) {
	var tried []string
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		tried = append(tried, address)
		if address == "kafka2:9092" {
			c1, c2 := net.Pipe()
			defer c2.Close()
			return c1, nil
		}
		return nil, errors.New("connection refused")
	}

	err := WaitForBroker(context.Background(), []string{"kafka1:9092", "kafka2:9092"}, 3, time.Millisecond, dial)
	if err != nil {
		t.Fatalf("WaitForBroker returned error: %v", err)
	}
	if len(tried) != 2 || tried[0] != "kafka1:9092" || tried[1] != "kafka2:9092" {
		t.Errorf("tried brokers = %v, want [kafka1:9092 kafka2:9092]", tried)
	}
}

func TestWaitForBrokerRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		cancel()
		return nil, errors.New("connection refused")
	}

	err := WaitForBroker(ctx, []string{"kafka:9092"}, 10, time.Hour, dial)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("WaitForBroker error = %v, want context.Canceled", err)
	}
}

// fakeReader serves a fixed list of messages and reports commits on a channel.
type fakeReader struct {
	messages []kafkago.Message
	fetched  int
	commitCh chan kafkago.Message
}

func newFakeReader(messages ...kafkago.Message) *fakeReader {
	return &fakeReader{messages: messages, commitCh: make(chan kafkago.Message, len(messages))}
}

func (f *fakeReader) FetchMessage(ctx context.Context) (kafkago.Message, error) {
	if f.fetched >= len(f.messages) {
		<-ctx.Done()
		return kafkago.Message{}, ctx.Err()
	}
	msg := f.messages[f.fetched]
	f.fetched++
	return msg, nil
}

func (f *fakeReader) CommitMessages(ctx context.Context, msgs ...kafkago.Message) error {
	for _, m := range msgs {
		f.commitCh <- m
	}
	return nil
}

func (f *fakeReader) Close() error { return nil }

// runConsumer runs a consumer over the fake reader until all messages are
// committed, then cancels it. It returns the captured log output.
func runConsumer(t *testing.T, reader *fakeReader, formatter MessageFormatter) string {
	t.Helper()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	consumer := newConsumer(reader, "movie-events", formatter)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	defer func() {
		cancel()
		<-done
	}()

	for range reader.messages {
		select {
		case <-reader.commitCh:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for message commits")
		}
	}
	return buf.String()
}

func TestConsumerAppliesFormatter(t *testing.T) {
	reader := newFakeReader(
		kafkago.Message{
			Topic:   "movie-events",
			Key:     []byte("42"),
			Value:   []byte("payload-1"),
			Headers: []kafkago.Header{{Key: "eventType", Value: []byte("MovieEvent")}},
		},
	)
	var gotKey, gotValue []byte
	var gotHeaders map[string]string
	formatter := func(key, value []byte, headers map[string]string) (string, error) {
		gotKey, gotValue, gotHeaders = key, value, headers
		return "FORMATTED", nil
	}

	out := runConsumer(t, reader, formatter)

	if string(gotKey) != "42" || string(gotValue) != "payload-1" {
		t.Errorf("formatter got key=%q value=%q, want key=%q value=%q", gotKey, gotValue, "42", "payload-1")
	}
	if gotHeaders["eventType"] != "MovieEvent" {
		t.Errorf("formatter got headers %v, want eventType=MovieEvent", gotHeaders)
	}
	if !strings.Contains(out, "consumed topic=movie-events") || !strings.Contains(out, "FORMATTED") {
		t.Errorf("log output %q does not contain the formatted message", out)
	}
}

func TestConsumerSurvivesFormatterError(t *testing.T) {
	reader := newFakeReader(
		kafkago.Message{Topic: "movie-events", Offset: 0, Key: []byte("1"), Value: []byte("bad")},
		kafkago.Message{Topic: "movie-events", Offset: 1, Key: []byte("2"), Value: []byte("good")},
	)
	formatter := func(key, value []byte, headers map[string]string) (string, error) {
		if string(value) == "bad" {
			return "", errors.New("decode failed")
		}
		return "OK:" + string(value), nil
	}

	// Both messages must be committed: a formatter error is logged and
	// skipped, the loop keeps running (runConsumer waits for all commits)
	out := runConsumer(t, reader, formatter)

	if !strings.Contains(out, "failed to format message") {
		t.Errorf("log output %q does not contain the formatter error", out)
	}
	if !strings.Contains(out, "OK:good") {
		t.Errorf("log output %q does not contain the message after the bad one", out)
	}
}
