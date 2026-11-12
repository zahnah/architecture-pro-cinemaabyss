package events

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventsv1 "events-service/internal/gen/events/v1"
)

// eventType values of the Kafka eventType header: proto message names
// without the package, see schemas/api/proto/cinemaabyss/events/v1/events.proto.
const (
	eventTypeMovie   = "MovieEvent"
	eventTypeUser    = "UserEvent"
	eventTypePayment = "PaymentEvent"
)

// registry maps eventType header values to prototypes for decoding.
var registry = map[string]func() proto.Message{
	eventTypeMovie:   func() proto.Message { return &eventsv1.MovieEvent{} },
	eventTypeUser:    func() proto.Message { return &eventsv1.UserEvent{} },
	eventTypePayment: func() proto.Message { return &eventsv1.PaymentEvent{} },
}

// Encode converts a domain event DTO into its Kafka representation: the
// target topic, the entity id as the key, the binary Protobuf payload and
// the eventType header value. A fresh event_id (UUID) and occurred_at
// (now) are stamped on every call.
func Encode(event Event) (topic, key string, payload []byte, eventType string, err error) {
	var msg proto.Message
	switch e := event.(type) {
	case MovieEvent:
		msg = &eventsv1.MovieEvent{
			EventId:    uuid.NewString(),
			OccurredAt: timestamppb.Now(),
			MovieId:    int64(e.MovieID),
			Title:      e.Title,
			Action:     e.Action,
			UserId:     int64(e.UserID),
		}
		key = strconv.Itoa(e.MovieID)
		eventType = eventTypeMovie
	case UserEvent:
		msg = &eventsv1.UserEvent{
			EventId:    uuid.NewString(),
			OccurredAt: timestamppb.Now(),
			UserId:     int64(e.UserID),
			Username:   e.Username,
			Action:     e.Action,
			Timestamp:  e.Timestamp,
		}
		key = strconv.Itoa(e.UserID)
		eventType = eventTypeUser
	case PaymentEvent:
		msg = &eventsv1.PaymentEvent{
			EventId:    uuid.NewString(),
			OccurredAt: timestamppb.Now(),
			PaymentId:  int64(e.PaymentID),
			UserId:     int64(e.UserID),
			Amount:     e.Amount,
			Status:     e.Status,
			Timestamp:  e.Timestamp,
			MethodType: e.MethodType,
		}
		key = strconv.Itoa(e.PaymentID)
		eventType = eventTypePayment
	default:
		return "", "", nil, "", fmt.Errorf("unknown event type %T", event)
	}

	payload, err = proto.Marshal(msg)
	if err != nil {
		return "", "", nil, "", fmt.Errorf("marshal %s: %w", eventType, err)
	}
	return event.Topic(), key, payload, eventType, nil
}

// Decode parses a binary Protobuf payload according to the eventType
// header value. Unknown event types are rejected.
func Decode(eventType string, data []byte) (proto.Message, error) {
	newMsg, ok := registry[eventType]
	if !ok {
		return nil, fmt.Errorf("unknown eventType %q", eventType)
	}
	msg := newMsg()
	if err := proto.Unmarshal(data, msg); err != nil {
		return nil, fmt.Errorf("unmarshal %s: %w", eventType, err)
	}
	return msg, nil
}
