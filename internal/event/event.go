package event

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType represents different types of events
type EventType string

const (
	UserCreatedEvent          EventType = "user.created"
	UserUpdatedEvent          EventType = "user.updated"
	UserDeletedEvent          EventType = "user.deleted"
	TransactionCreatedEvent   EventType = "transaction.created"
	TransactionCompletedEvent EventType = "transaction.completed"
	TransactionFailedEvent    EventType = "transaction.failed"
	TransactionCancelledEvent EventType = "transaction.cancelled"
	BalanceCreditedEvent      EventType = "balance.credited"
	BalanceDebitedEvent       EventType = "balance.debited"
)

// Event represents a domain event
type Event struct {
	ID          uuid.UUID       `json:"id"`
	Type        EventType       `json:"type"`
	AggregateID uuid.UUID       `json:"aggregate_id"`
	Data        json.RawMessage `json:"data"`
	Metadata    Metadata        `json:"metadata"`
	Version     int             `json:"version"`
	CreatedAt   time.Time       `json:"created_at"`
}

// Metadata contains additional information about the event
type Metadata struct {
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	IPAddress string     `json:"ip_address,omitempty"`
	UserAgent string     `json:"user_agent,omitempty"`
	Source    string     `json:"source,omitempty"`
}

// NewEvent creates a new event
func NewEvent(eventType EventType, aggregateID uuid.UUID, data interface{}, metadata Metadata, version int) (*Event, error) {
	eventData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Event{
		ID:          uuid.New(),
		Type:        eventType,
		AggregateID: aggregateID,
		Data:        eventData,
		Metadata:    metadata,
		Version:     version,
		CreatedAt:   time.Now(),
	}, nil
}

// GetData unmarshals event data into the provided interface
func (e *Event) GetData(dest interface{}) error {
	return json.Unmarshal(e.Data, dest)
}

// EventHandler defines the interface for handling events
type EventHandler interface {
	Handle(event *Event) error
	EventTypes() []EventType
}

// EventStore defines the interface for storing and retrieving events
type EventStore interface {
	SaveEvent(event *Event) error
	GetEvents(aggregateID uuid.UUID) ([]*Event, error)
	GetEventsByType(eventType EventType, limit, offset int) ([]*Event, error)
	GetEventsAfter(timestamp time.Time, limit int) ([]*Event, error)
	GetLastEventVersion(aggregateID uuid.UUID) (int, error)
}

// EventBus defines the interface for publishing and subscribing to events
type EventBus interface {
	Publish(event *Event) error
	Subscribe(eventType EventType, handler EventHandler) error
	Unsubscribe(eventType EventType, handler EventHandler) error
}

// EventStream represents a stream of events
type EventStream struct {
	Events    []*Event  `json:"events"`
	Version   int       `json:"version"`
	StreamID  string    `json:"stream_id"`
	CreatedAt time.Time `json:"created_at"`
}

// EventData definitions for different event types

// UserCreatedEventData represents data for user created events
type UserCreatedEventData struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
}

// UserUpdatedEventData represents data for user updated events
type UserUpdatedEventData struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	OldUsername string    `json:"old_username"`
	OldEmail    string    `json:"old_email"`
	OldRole     string    `json:"old_role"`
}

// UserDeletedEventData represents data for user deleted events
type UserDeletedEventData struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
}

// TransactionCreatedEventData represents data for transaction created events
type TransactionCreatedEventData struct {
	TransactionID uuid.UUID  `json:"transaction_id"`
	FromUserID    *uuid.UUID `json:"from_user_id"`
	ToUserID      *uuid.UUID `json:"to_user_id"`
	Amount        float64    `json:"amount"`
	Type          string     `json:"type"`
	Status        string     `json:"status"`
	Description   string     `json:"description"`
	ReferenceID   string     `json:"reference_id"`
}

// TransactionStatusChangedEventData represents data for transaction status change events
type TransactionStatusChangedEventData struct {
	TransactionID uuid.UUID `json:"transaction_id"`
	OldStatus     string    `json:"old_status"`
	NewStatus     string    `json:"new_status"`
	Reason        string    `json:"reason,omitempty"`
}

// BalanceChangedEventData represents data for balance change events
type BalanceChangedEventData struct {
	UserID        uuid.UUID  `json:"user_id"`
	OldBalance    float64    `json:"old_balance"`
	NewBalance    float64    `json:"new_balance"`
	Amount        float64    `json:"amount"`
	Operation     string     `json:"operation"`
	TransactionID *uuid.UUID `json:"transaction_id,omitempty"`
}

// Snapshot represents a point-in-time snapshot of an aggregate
type Snapshot struct {
	AggregateID   uuid.UUID       `json:"aggregate_id"`
	AggregateType string          `json:"aggregate_type"`
	Data          json.RawMessage `json:"data"`
	Version       int             `json:"version"`
	CreatedAt     time.Time       `json:"created_at"`
}

// NewSnapshot creates a new snapshot
func NewSnapshot(aggregateID uuid.UUID, aggregateType string, data interface{}, version int) (*Snapshot, error) {
	snapshotData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Snapshot{
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Data:          snapshotData,
		Version:       version,
		CreatedAt:     time.Now(),
	}, nil
}

// GetData unmarshals snapshot data into the provided interface
func (s *Snapshot) GetData(dest interface{}) error {
	return json.Unmarshal(s.Data, dest)
}

// SnapshotStore defines the interface for storing and retrieving snapshots
type SnapshotStore interface {
	SaveSnapshot(snapshot *Snapshot) error
	GetSnapshot(aggregateID uuid.UUID) (*Snapshot, error)
	DeleteSnapshot(aggregateID uuid.UUID) error
}

// ProjectionHandler defines the interface for handling projections
type ProjectionHandler interface {
	Handle(event *Event) error
	Rebuild() error
	GetName() string
}

// EventFilter defines criteria for filtering events
type EventFilter struct {
	AggregateID *uuid.UUID  `json:"aggregate_id,omitempty"`
	EventTypes  []EventType `json:"event_types,omitempty"`
	FromTime    *time.Time  `json:"from_time,omitempty"`
	ToTime      *time.Time  `json:"to_time,omitempty"`
	Limit       int         `json:"limit,omitempty"`
	Offset      int         `json:"offset,omitempty"`
}

// EventQuery represents a query for events
type EventQuery struct {
	Filter EventFilter `json:"filter"`
	SortBy string      `json:"sort_by"`
	Order  string      `json:"order"`
}

// EventReplay allows replaying events for rebuilding projections
type EventReplay struct {
	FromVersion *int        `json:"from_version,omitempty"`
	ToVersion   *int        `json:"to_version,omitempty"`
	FromTime    *time.Time  `json:"from_time,omitempty"`
	ToTime      *time.Time  `json:"to_time,omitempty"`
	EventTypes  []EventType `json:"event_types,omitempty"`
	BatchSize   int         `json:"batch_size"`
}

// EventPublisher defines the interface for publishing events to external systems
type EventPublisher interface {
	Publish(event *Event) error
	PublishBatch(events []*Event) error
}

// EventSubscriber defines the interface for subscribing to events from external systems
type EventSubscriber interface {
	Subscribe(eventTypes []EventType, handler func(*Event) error) error
	Unsubscribe() error
	Start() error
	Stop() error
}
