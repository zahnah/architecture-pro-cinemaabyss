package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"events-service/internal/events"
)

// mockProducer records published messages and can be configured to fail.
type mockProducer struct {
	topic   string
	key     string
	payload []byte
	headers map[string]string
	err     error
}

func (m *mockProducer) Publish(ctx context.Context, topic, key string, payload []byte, headers map[string]string) error {
	if m.err != nil {
		return m.err
	}
	m.topic = topic
	m.key = key
	m.payload = payload
	m.headers = headers
	return nil
}

func TestHealth(t *testing.T) {
	h := New(&mockProducer{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/events/health", nil)

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("health status = %d, want %d", rec.Code, http.StatusOK)
	}
	if body := rec.Body.String(); body != "{\"status\":true}\n" {
		t.Errorf("health body = %q, want %q", body, "{\"status\":true}\n")
	}
}

func TestCreateEvents(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		body          string
		wantTopic     string
		wantKey       string
		wantEventType string
	}{
		{
			name:          "movie event",
			path:          "/api/events/movie",
			body:          `{"movie_id":1,"title":"Test Movie Event","action":"viewed","user_id":1}`,
			wantTopic:     events.TopicMovieEvents,
			wantKey:       "1",
			wantEventType: "MovieEvent",
		},
		{
			name:          "user event",
			path:          "/api/events/user",
			body:          `{"user_id":2,"username":"testuser","action":"logged_in","timestamp":"2026-11-11T00:00:00.000Z"}`,
			wantTopic:     events.TopicUserEvents,
			wantKey:       "2",
			wantEventType: "UserEvent",
		},
		{
			name:          "payment event",
			path:          "/api/events/payment",
			body:          `{"payment_id":3,"user_id":2,"amount":9.99,"status":"completed","timestamp":"2026-11-11T00:00:00.000Z","method_type":"credit_card"}`,
			wantTopic:     events.TopicPaymentEvents,
			wantKey:       "3",
			wantEventType: "PaymentEvent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &mockProducer{}
			h := New(p)
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))

			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusCreated {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
			}
			if body := rec.Body.String(); body != "{\"status\":\"success\"}\n" {
				t.Errorf("body = %q, want %q", body, "{\"status\":\"success\"}\n")
			}
			if p.topic != tt.wantTopic {
				t.Errorf("published to topic %q, want %q", p.topic, tt.wantTopic)
			}
			if p.key != tt.wantKey {
				t.Errorf("published with key %q, want %q", p.key, tt.wantKey)
			}
			if got := p.headers["eventType"]; got != tt.wantEventType {
				t.Errorf("eventType header = %q, want %q", got, tt.wantEventType)
			}
			// The published payload must decode as the declared event type
			if _, err := events.Decode(tt.wantEventType, p.payload); err != nil {
				t.Errorf("published payload does not decode as %s: %v", tt.wantEventType, err)
			}
		})
	}
}

func TestCreateEventInvalidJSON(t *testing.T) {
	h := New(&mockProducer{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events/movie", strings.NewReader(`{invalid`))

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateEventProducerError(t *testing.T) {
	p := &mockProducer{err: errors.New("kafka unavailable")}
	h := New(p)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/events/movie",
		strings.NewReader(`{"movie_id":1,"title":"T","action":"viewed","user_id":1}`))

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestUnknownPathReturns404(t *testing.T) {
	h := New(&mockProducer{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/events/unknown", nil)

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
