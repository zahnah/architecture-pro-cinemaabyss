package events

import (
	"testing"

	eventsv1 "events-service/internal/gen/events/v1"
)

func TestEncodeDecodeRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		key   string
		topic string
		check func(t *testing.T, msg any)
	}{
		{
			name:  "movie event",
			event: MovieEvent{MovieID: 42, Title: "T", Action: "viewed", UserID: 7},
			key:   "42",
			topic: TopicMovieEvents,
			check: func(t *testing.T, msg any) {
				m, ok := msg.(*eventsv1.MovieEvent)
				if !ok {
					t.Fatalf("decoded type = %T, want *eventsv1.MovieEvent", msg)
				}
				if m.MovieId != 42 || m.Title != "T" || m.Action != "viewed" || m.UserId != 7 {
					t.Errorf("unexpected movie event: %+v", m)
				}
			},
		},
		{
			name:  "user event",
			event: UserEvent{UserID: 7, Username: "testuser", Action: "logged_in", Timestamp: "2026-11-11T00:00:00.000Z"},
			key:   "7",
			topic: TopicUserEvents,
			check: func(t *testing.T, msg any) {
				u, ok := msg.(*eventsv1.UserEvent)
				if !ok {
					t.Fatalf("decoded type = %T, want *eventsv1.UserEvent", msg)
				}
				if u.UserId != 7 || u.Username != "testuser" || u.Action != "logged_in" ||
					u.Timestamp != "2026-11-11T00:00:00.000Z" {
					t.Errorf("unexpected user event: %+v", u)
				}
			},
		},
		{
			name: "payment event",
			event: PaymentEvent{PaymentID: 9, UserID: 7, Amount: 9.99, Status: "completed",
				Timestamp: "2026-11-11T00:00:00.000Z", MethodType: "credit_card"},
			key:   "9",
			topic: TopicPaymentEvents,
			check: func(t *testing.T, msg any) {
				p, ok := msg.(*eventsv1.PaymentEvent)
				if !ok {
					t.Fatalf("decoded type = %T, want *eventsv1.PaymentEvent", msg)
				}
				if p.PaymentId != 9 || p.UserId != 7 || p.Amount != 9.99 || p.Status != "completed" ||
					p.Timestamp != "2026-11-11T00:00:00.000Z" || p.MethodType != "credit_card" {
					t.Errorf("unexpected payment event: %+v", p)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic, key, payload, eventType, err := Encode(tt.event)
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			if topic != tt.topic {
				t.Errorf("topic = %q, want %q", topic, tt.topic)
			}
			if key != tt.key {
				t.Errorf("key = %q, want %q", key, tt.key)
			}
			if eventType == "" {
				t.Error("eventType is empty")
			}

			msg, err := Decode(eventType, payload)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			tt.check(t, msg)
		})
	}
}

func TestEncodeStampsEventIDAndOccurredAt(t *testing.T) {
	_, _, payload1, eventType, err := Encode(MovieEvent{MovieID: 1, UserID: 1})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	_, _, payload2, _, err := Encode(MovieEvent{MovieID: 1, UserID: 1})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	msg1, err := Decode(eventType, payload1)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	msg2, err := Decode(eventType, payload2)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	e1 := msg1.(*eventsv1.MovieEvent)
	e2 := msg2.(*eventsv1.MovieEvent)

	if e1.EventId == "" {
		t.Error("event_id is empty")
	}
	if e1.EventId == e2.EventId {
		t.Errorf("event_id is not unique: %q", e1.EventId)
	}
	if e1.OccurredAt == nil {
		t.Error("occurred_at is not set")
	} else if err := e1.OccurredAt.CheckValid(); err != nil {
		t.Errorf("occurred_at is invalid: %v", err)
	}
}

func TestEncodeUnknownEventType(t *testing.T) {
	if _, _, _, _, err := Encode(nil); err == nil {
		t.Error("Encode(nil) returned nil error, want error")
	}
}

func TestDecodeUnknownEventType(t *testing.T) {
	if _, err := Decode("NoSuchEvent", []byte{}); err == nil {
		t.Error("Decode with unknown eventType returned nil error, want error")
	}
}

func TestDecodeCorruptPayload(t *testing.T) {
	if _, err := Decode(eventTypeMovie, []byte{0xff, 0xff, 0xff}); err == nil {
		t.Error("Decode with corrupt payload returned nil error, want error")
	}
}
