// Package events defines the domain events of the events service and their
// mapping to Kafka topics.
package events

// Kafka topics of the events service.
const (
	TopicMovieEvents   = "movie-events"
	TopicUserEvents    = "user-events"
	TopicPaymentEvents = "payment-events"
)

// Topics returns all Kafka topics of the service.
func Topics() []string {
	return []string{TopicMovieEvents, TopicUserEvents, TopicPaymentEvents}
}

// Event is a domain event routed to a Kafka topic.
type Event interface {
	Topic() string
}

// MovieEvent is emitted on movie-related actions (viewed, rated, ...).
type MovieEvent struct {
	MovieID int    `json:"movie_id"`
	Title   string `json:"title"`
	Action  string `json:"action"`
	UserID  int    `json:"user_id"`
}

// Topic returns the Kafka topic for movie events.
func (MovieEvent) Topic() string { return TopicMovieEvents }

// UserEvent is emitted on user-related actions (registered, logged_in, ...).
type UserEvent struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Action    string `json:"action"`
	Timestamp string `json:"timestamp"`
}

// Topic returns the Kafka topic for user events.
func (UserEvent) Topic() string { return TopicUserEvents }

// PaymentEvent is emitted on payment-related actions.
type PaymentEvent struct {
	PaymentID  int     `json:"payment_id"`
	UserID     int     `json:"user_id"`
	Amount     float64 `json:"amount"`
	Status     string  `json:"status"`
	Timestamp  string  `json:"timestamp"`
	MethodType string  `json:"method_type"`
}

// Topic returns the Kafka topic for payment events.
func (PaymentEvent) Topic() string { return TopicPaymentEvents }
