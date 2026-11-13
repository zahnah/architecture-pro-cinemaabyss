// Package handler implements the HTTP API of the events service.
package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"events-service/internal/events"
	"events-service/internal/kafka"
)

// Events is the HTTP entry point of the events service: it accepts event
// creation requests and publishes them to Kafka.
type Events struct {
	producer kafka.Producer
}

// New creates an Events handler publishing via the given producer.
func New(producer kafka.Producer) *Events {
	return &Events{producer: producer}
}

func (e *Events) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/events/health":
		handleHealth(w)
	case r.Method == http.MethodPost && r.URL.Path == "/api/events/movie":
		publishEvent[events.MovieEvent](e.producer, w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/events/user":
		publishEvent[events.UserEvent](e.producer, w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/events/payment":
		publishEvent[events.PaymentEvent](e.producer, w, r)
	default:
		http.NotFound(w, r)
	}
}

func handleHealth(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"status": true})
}

func publishEvent[E events.Event](p kafka.Producer, w http.ResponseWriter, r *http.Request) {
	var event E
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	topic, key, payload, eventType, err := events.Encode(event)
	if err != nil {
		http.Error(w, "failed to encode event", http.StatusInternalServerError)
		return
	}
	headers := map[string]string{"eventType": eventType}
	if err := p.Publish(r.Context(), topic, key, payload, headers); err != nil {
		log.Printf("Failed to publish event to topic %s: %v", topic, err)
		http.Error(w, "failed to publish event", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
