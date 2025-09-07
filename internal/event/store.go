package event

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// PostgresEventStore implements EventStore using PostgreSQL
type PostgresEventStore struct {
	db *sql.DB
}

// NewPostgresEventStore creates a new PostgreSQL event store
func NewPostgresEventStore(db *sql.DB) *PostgresEventStore {
	return &PostgresEventStore{db: db}
}

// SaveEvent saves an event to the store
func (s *PostgresEventStore) SaveEvent(event *Event) error {
	query := `
		INSERT INTO events (id, type, aggregate_id, data, metadata, version, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.db.Exec(query,
		event.ID,
		event.Type,
		event.AggregateID,
		event.Data,
		metadataJSON,
		event.Version,
		event.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save event: %w", err)
	}

	log.Debug().
		Str("event_id", event.ID.String()).
		Str("event_type", string(event.Type)).
		Str("aggregate_id", event.AggregateID.String()).
		Int("version", event.Version).
		Msg("Event saved")

	return nil
}

// GetEvents retrieves all events for a specific aggregate
func (s *PostgresEventStore) GetEvents(aggregateID uuid.UUID) ([]*Event, error) {
	query := `
		SELECT id, type, aggregate_id, data, metadata, version, created_at
		FROM events 
		WHERE aggregate_id = $1
		ORDER BY version ASC`

	rows, err := s.db.Query(query, aggregateID)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		var metadataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.AggregateID,
			&event.Data,
			&metadataJSON,
			&event.Version,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		events = append(events, event)
	}

	return events, nil
}

// GetEventsByType retrieves events by type with pagination
func (s *PostgresEventStore) GetEventsByType(eventType EventType, limit, offset int) ([]*Event, error) {
	query := `
		SELECT id, type, aggregate_id, data, metadata, version, created_at
		FROM events 
		WHERE type = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := s.db.Query(query, eventType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by type: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		var metadataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.AggregateID,
			&event.Data,
			&metadataJSON,
			&event.Version,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		events = append(events, event)
	}

	return events, nil
}

// GetEventsAfter retrieves events created after a specific timestamp
func (s *PostgresEventStore) GetEventsAfter(timestamp time.Time, limit int) ([]*Event, error) {
	query := `
		SELECT id, type, aggregate_id, data, metadata, version, created_at
		FROM events 
		WHERE created_at > $1
		ORDER BY created_at ASC
		LIMIT $2`

	rows, err := s.db.Query(query, timestamp, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query events after timestamp: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		var metadataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&event.AggregateID,
			&event.Data,
			&metadataJSON,
			&event.Version,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		events = append(events, event)
	}

	return events, nil
}

// GetLastEventVersion retrieves the last event version for an aggregate
func (s *PostgresEventStore) GetLastEventVersion(aggregateID uuid.UUID) (int, error) {
	query := `
		SELECT COALESCE(MAX(version), 0) 
		FROM events 
		WHERE aggregate_id = $1`

	var version int
	err := s.db.QueryRow(query, aggregateID).Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get last event version: %w", err)
	}

	return version, nil
}

// InMemoryEventBus implements EventBus using in-memory storage
type InMemoryEventBus struct {
	handlers map[EventType][]EventHandler
}

// NewInMemoryEventBus creates a new in-memory event bus
func NewInMemoryEventBus() *InMemoryEventBus {
	return &InMemoryEventBus{
		handlers: make(map[EventType][]EventHandler),
	}
}

// Publish publishes an event to all registered handlers
func (b *InMemoryEventBus) Publish(event *Event) error {
	handlers, exists := b.handlers[event.Type]
	if !exists {
		log.Debug().
			Str("event_type", string(event.Type)).
			Msg("No handlers registered for event type")
		return nil
	}

	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h.Handle(event); err != nil {
				log.Error().
					Err(err).
					Str("event_id", event.ID.String()).
					Str("event_type", string(event.Type)).
					Msg("Failed to handle event")
			}
		}(handler)
	}

	log.Debug().
		Str("event_id", event.ID.String()).
		Str("event_type", string(event.Type)).
		Int("handlers", len(handlers)).
		Msg("Event published")

	return nil
}

// Subscribe registers a handler for specific event types
func (b *InMemoryEventBus) Subscribe(eventType EventType, handler EventHandler) error {
	if b.handlers[eventType] == nil {
		b.handlers[eventType] = make([]EventHandler, 0)
	}

	b.handlers[eventType] = append(b.handlers[eventType], handler)

	log.Info().
		Str("event_type", string(eventType)).
		Msg("Handler subscribed")

	return nil
}

// Unsubscribe removes a handler for specific event types
func (b *InMemoryEventBus) Unsubscribe(eventType EventType, handler EventHandler) error {
	handlers, exists := b.handlers[eventType]
	if !exists {
		return nil
	}

	for i, h := range handlers {
		if h == handler {
			b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			break
		}
	}

	log.Info().
		Str("event_type", string(eventType)).
		Msg("Handler unsubscribed")

	return nil
}

// PostgresSnapshotStore implements SnapshotStore using PostgreSQL
type PostgresSnapshotStore struct {
	db *sql.DB
}

// NewPostgresSnapshotStore creates a new PostgreSQL snapshot store
func NewPostgresSnapshotStore(db *sql.DB) *PostgresSnapshotStore {
	return &PostgresSnapshotStore{db: db}
}

// SaveSnapshot saves a snapshot to the store
func (s *PostgresSnapshotStore) SaveSnapshot(snapshot *Snapshot) error {
	query := `
		INSERT INTO snapshots (aggregate_id, aggregate_type, data, version, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (aggregate_id) 
		DO UPDATE SET 
			aggregate_type = EXCLUDED.aggregate_type,
			data = EXCLUDED.data,
			version = EXCLUDED.version,
			created_at = EXCLUDED.created_at`

	_, err := s.db.Exec(query,
		snapshot.AggregateID,
		snapshot.AggregateType,
		snapshot.Data,
		snapshot.Version,
		snapshot.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save snapshot: %w", err)
	}

	log.Debug().
		Str("aggregate_id", snapshot.AggregateID.String()).
		Str("aggregate_type", snapshot.AggregateType).
		Int("version", snapshot.Version).
		Msg("Snapshot saved")

	return nil
}

// GetSnapshot retrieves a snapshot for an aggregate
func (s *PostgresSnapshotStore) GetSnapshot(aggregateID uuid.UUID) (*Snapshot, error) {
	query := `
		SELECT aggregate_id, aggregate_type, data, version, created_at
		FROM snapshots 
		WHERE aggregate_id = $1`

	snapshot := &Snapshot{}
	err := s.db.QueryRow(query, aggregateID).Scan(
		&snapshot.AggregateID,
		&snapshot.AggregateType,
		&snapshot.Data,
		&snapshot.Version,
		&snapshot.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("snapshot not found")
		}
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	return snapshot, nil
}

// DeleteSnapshot deletes a snapshot for an aggregate
func (s *PostgresSnapshotStore) DeleteSnapshot(aggregateID uuid.UUID) error {
	query := `DELETE FROM snapshots WHERE aggregate_id = $1`

	result, err := s.db.Exec(query, aggregateID)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("snapshot not found")
	}

	log.Debug().
		Str("aggregate_id", aggregateID.String()).
		Msg("Snapshot deleted")

	return nil
}

// EventService provides high-level event operations
type EventService struct {
	store EventStore
	bus   EventBus
}

// NewEventService creates a new event service
func NewEventService(store EventStore, bus EventBus) *EventService {
	return &EventService{
		store: store,
		bus:   bus,
	}
}

// PublishAndStore publishes an event and stores it
func (s *EventService) PublishAndStore(event *Event) error {
	// Store the event first
	if err := s.store.SaveEvent(event); err != nil {
		return fmt.Errorf("failed to store event: %w", err)
	}

	// Then publish it
	if err := s.bus.Publish(event); err != nil {
		log.Error().
			Err(err).
			Str("event_id", event.ID.String()).
			Msg("Failed to publish event, but it was stored")
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// ReplayEvents replays events for rebuilding projections
func (s *EventService) ReplayEvents(ctx context.Context, replay EventReplay, handler func(*Event) error) error {
	var events []*Event
	var err error

	// Get events based on replay criteria
	if replay.FromTime != nil {
		events, err = s.store.GetEventsAfter(*replay.FromTime, replay.BatchSize)
	} else {
		// For simplicity, get all events (in a real implementation, you'd want pagination)
		return fmt.Errorf("replay without time criteria not implemented")
	}

	if err != nil {
		return fmt.Errorf("failed to get events for replay: %w", err)
	}

	// Process events in batches
	for _, event := range events {
		// Filter by event types if specified
		if len(replay.EventTypes) > 0 {
			found := false
			for _, eventType := range replay.EventTypes {
				if event.Type == eventType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by version if specified
		if replay.FromVersion != nil && event.Version < *replay.FromVersion {
			continue
		}
		if replay.ToVersion != nil && event.Version > *replay.ToVersion {
			continue
		}

		// Handle the event
		if err := handler(event); err != nil {
			log.Error().
				Err(err).
				Str("event_id", event.ID.String()).
				Msg("Failed to handle event during replay")
			return fmt.Errorf("failed to handle event during replay: %w", err)
		}

		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	log.Info().
		Int("events_processed", len(events)).
		Msg("Event replay completed")

	return nil
}
