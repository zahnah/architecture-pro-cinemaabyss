package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"

	"events-service/internal/config"
	"events-service/internal/events"
	"events-service/internal/handler"
	"events-service/internal/kafka"
)

// formatMessage renders a consumed Kafka message as readable JSON: the
// binary Protobuf payload is decoded by its eventType header.
func formatMessage(key, value []byte, headers map[string]string) (string, error) {
	eventType := headers["eventType"]
	msg, err := events.Decode(eventType, value)
	if err != nil {
		return "", err
	}
	jsonBytes, err := protojson.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("marshal %s to json: %w", eventType, err)
	}
	return fmt.Sprintf("key=%s eventType=%s value=%s", key, eventType, jsonBytes), nil
}

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := kafka.WaitForBroker(ctx, cfg.KafkaBrokers, cfg.RetryAttempts, cfg.RetryInterval, nil); err != nil {
		log.Fatalf("Kafka is not reachable: %v", err)
	}

	producer := kafka.NewProducer(cfg.KafkaBrokers)
	defer producer.Close()

	for _, topic := range events.Topics() {
		consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.ConsumerGroup, topic, formatMessage)
		go func() {
			log.Printf("Starting consumer for topic %s (group %s)", topic, cfg.ConsumerGroup)
			if err := consumer.Run(ctx); err != nil && ctx.Err() == nil {
				log.Printf("Consumer for topic %s stopped: %v", topic, err)
			}
		}()
	}

	log.Printf("Starting events service on port %s (kafka brokers: %v)", cfg.Port, cfg.KafkaBrokers)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, handler.New(producer)))
}
