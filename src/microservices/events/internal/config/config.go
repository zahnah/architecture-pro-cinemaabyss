package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the events-service configuration loaded from environment.
type Config struct {
	Port          string
	KafkaBrokers  []string
	ConsumerGroup string
	RetryAttempts int
	RetryInterval time.Duration
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Load reads configuration from environment variables, falling back to
// defaults and logging about invalid values.
func Load() Config {
	return Config{
		Port:          getEnv("PORT", "8082"),
		KafkaBrokers:  splitBrokers(getEnv("KAFKA_BROKERS", "kafka:9092")),
		ConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "events-service"),
		RetryAttempts: parseInt(getEnv("KAFKA_RETRY_ATTEMPTS", "30"), 30),
		RetryInterval: parseDuration(getEnv("KAFKA_RETRY_INTERVAL", "2s"), 2*time.Second),
	}
}

// splitBrokers splits a comma-separated broker list, trimming whitespace
// and dropping empty entries.
func splitBrokers(raw string) []string {
	var brokers []string
	for _, b := range strings.Split(raw, ",") {
		if b = strings.TrimSpace(b); b != "" {
			brokers = append(brokers, b)
		}
	}
	return brokers
}

func parseInt(raw string, fallback int) int {
	v, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("Invalid integer value %q, defaulting to %d", raw, fallback)
		return fallback
	}
	return v
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	v, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("Invalid duration value %q, defaulting to %s", raw, fallback)
		return fallback
	}
	return v
}
