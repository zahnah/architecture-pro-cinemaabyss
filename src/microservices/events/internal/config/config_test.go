package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Empty values exercise the fallback path of getEnv
	t.Setenv("PORT", "")
	t.Setenv("KAFKA_BROKERS", "")
	t.Setenv("KAFKA_CONSUMER_GROUP", "")
	t.Setenv("KAFKA_RETRY_ATTEMPTS", "")
	t.Setenv("KAFKA_RETRY_INTERVAL", "")

	cfg := Load()

	if cfg.Port != "8082" {
		t.Errorf("Port = %q, want %q", cfg.Port, "8082")
	}
	if !reflect.DeepEqual(cfg.KafkaBrokers, []string{"kafka:9092"}) {
		t.Errorf("KafkaBrokers = %v, want %v", cfg.KafkaBrokers, []string{"kafka:9092"})
	}
	if cfg.ConsumerGroup != "events-service" {
		t.Errorf("ConsumerGroup = %q, want %q", cfg.ConsumerGroup, "events-service")
	}
	if cfg.RetryAttempts != 30 {
		t.Errorf("RetryAttempts = %d, want 30", cfg.RetryAttempts)
	}
	if cfg.RetryInterval != 2*time.Second {
		t.Errorf("RetryInterval = %s, want 2s", cfg.RetryInterval)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("KAFKA_BROKERS", "kafka1:9092, kafka2:9093")
	t.Setenv("KAFKA_CONSUMER_GROUP", "custom-group")
	t.Setenv("KAFKA_RETRY_ATTEMPTS", "5")
	t.Setenv("KAFKA_RETRY_INTERVAL", "500ms")

	cfg := Load()

	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if !reflect.DeepEqual(cfg.KafkaBrokers, []string{"kafka1:9092", "kafka2:9093"}) {
		t.Errorf("KafkaBrokers = %v, want %v", cfg.KafkaBrokers, []string{"kafka1:9092", "kafka2:9093"})
	}
	if cfg.ConsumerGroup != "custom-group" {
		t.Errorf("ConsumerGroup = %q, want %q", cfg.ConsumerGroup, "custom-group")
	}
	if cfg.RetryAttempts != 5 {
		t.Errorf("RetryAttempts = %d, want 5", cfg.RetryAttempts)
	}
	if cfg.RetryInterval != 500*time.Millisecond {
		t.Errorf("RetryInterval = %s, want 500ms", cfg.RetryInterval)
	}
}

func TestLoadInvalidRetryValuesFallBack(t *testing.T) {
	t.Setenv("KAFKA_RETRY_ATTEMPTS", "not-a-number")
	t.Setenv("KAFKA_RETRY_INTERVAL", "not-a-duration")

	cfg := Load()

	if cfg.RetryAttempts != 30 {
		t.Errorf("RetryAttempts = %d, want 30", cfg.RetryAttempts)
	}
	if cfg.RetryInterval != 2*time.Second {
		t.Errorf("RetryInterval = %s, want 2s", cfg.RetryInterval)
	}
}

func TestSplitBrokers(t *testing.T) {
	tests := []struct {
		raw  string
		want []string
	}{
		{"kafka:9092", []string{"kafka:9092"}},
		{"kafka1:9092,kafka2:9093", []string{"kafka1:9092", "kafka2:9093"}},
		{" kafka1:9092 , kafka2:9093 ", []string{"kafka1:9092", "kafka2:9093"}},
		{"kafka1:9092,,kafka2:9093", []string{"kafka1:9092", "kafka2:9093"}},
		{"", nil},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := splitBrokers(tt.raw); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitBrokers(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}
