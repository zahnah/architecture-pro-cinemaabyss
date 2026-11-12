package events

import (
	"encoding/json"
	"testing"
)

// The payloads below are verbatim copies of the request bodies from
// tests/postman/CinemaAbyss.postman_collection.json (Events Microservice
// folder) and must keep decoding correctly.

func TestMovieEventUnmarshal(t *testing.T) {
	raw := `{
    "movie_id": 1,
    "title": "Test Movie Event",
    "action": "viewed",
    "user_id": 1
}`
	var e MovieEvent
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal movie event: %v", err)
	}
	if e.MovieID != 1 || e.Title != "Test Movie Event" || e.Action != "viewed" || e.UserID != 1 {
		t.Errorf("unexpected movie event: %+v", e)
	}
}

func TestUserEventUnmarshal(t *testing.T) {
	raw := `{
    "user_id": 1,
    "username": "testuser",
    "action": "logged_in",
    "timestamp": "2026-11-11T00:00:00.000Z"
}`
	var e UserEvent
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal user event: %v", err)
	}
	if e.UserID != 1 || e.Username != "testuser" || e.Action != "logged_in" || e.Timestamp != "2026-11-11T00:00:00.000Z" {
		t.Errorf("unexpected user event: %+v", e)
	}
}

func TestPaymentEventUnmarshal(t *testing.T) {
	raw := `{
    "payment_id": 1,
    "user_id": 1,
    "amount": 9.99,
    "status": "completed",
    "timestamp": "2026-11-11T00:00:00.000Z",
    "method_type": "credit_card"
}`
	var e PaymentEvent
	if err := json.Unmarshal([]byte(raw), &e); err != nil {
		t.Fatalf("unmarshal payment event: %v", err)
	}
	if e.PaymentID != 1 || e.UserID != 1 || e.Amount != 9.99 || e.Status != "completed" ||
		e.Timestamp != "2026-11-11T00:00:00.000Z" || e.MethodType != "credit_card" {
		t.Errorf("unexpected payment event: %+v", e)
	}
}

func TestEventTopicMapping(t *testing.T) {
	tests := []struct {
		event Event
		want  string
	}{
		{MovieEvent{}, TopicMovieEvents},
		{UserEvent{}, TopicUserEvents},
		{PaymentEvent{}, TopicPaymentEvents},
	}
	for _, tt := range tests {
		if got := tt.event.Topic(); got != tt.want {
			t.Errorf("%T.Topic() = %q, want %q", tt.event, got, tt.want)
		}
	}
}

func TestTopics(t *testing.T) {
	want := []string{TopicMovieEvents, TopicUserEvents, TopicPaymentEvents}
	got := Topics()
	if len(got) != len(want) {
		t.Fatalf("Topics() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Topics()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
