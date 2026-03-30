package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sebasing/event-driven-service/internal/models"
)

// PostgresStore provides database operations for onboarding flows and events.
type PostgresStore struct {
	pool   *pgxpool.Pool
	schema string
}

// NewPostgresStore creates a new store backed by the given connection pool.
func NewPostgresStore(pool *pgxpool.Pool, schema string) *PostgresStore {
	return &PostgresStore{pool: pool, schema: schema}
}

// CreateFlow inserts a new onboarding flow.
func (s *PostgresStore) CreateFlow(ctx context.Context, flow *models.OnboardingFlow) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.onboarding_flows (id, correlation_id, user_email, status, started_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, s.schema)

	_, err := s.pool.Exec(ctx, query,
		flow.ID, flow.CorrelationID, flow.UserEmail,
		flow.Status, flow.StartedAt, flow.CreatedAt,
	)
	return err
}

// GetFlowByCorrelationID retrieves a flow by its correlation ID.
func (s *PostgresStore) GetFlowByCorrelationID(ctx context.Context, correlationID uuid.UUID) (*models.OnboardingFlow, error) {
	query := fmt.Sprintf(`
		SELECT id, correlation_id, user_email, status, started_at, completed_at, created_at
		FROM %s.onboarding_flows
		WHERE correlation_id = $1
	`, s.schema)

	var flow models.OnboardingFlow
	err := s.pool.QueryRow(ctx, query, correlationID).Scan(
		&flow.ID, &flow.CorrelationID, &flow.UserEmail,
		&flow.Status, &flow.StartedAt, &flow.CompletedAt, &flow.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &flow, nil
}

// UpdateFlowStatus updates the status of a flow. If the status is "completed",
// it also sets the completed_at timestamp.
func (s *PostgresStore) UpdateFlowStatus(ctx context.Context, correlationID uuid.UUID, status string) error {
	var query string
	if status == models.FlowStatusCompleted {
		query = fmt.Sprintf(`
			UPDATE %s.onboarding_flows
			SET status = $1, completed_at = $2
			WHERE correlation_id = $3
		`, s.schema)
		_, err := s.pool.Exec(ctx, query, status, time.Now(), correlationID)
		return err
	}

	query = fmt.Sprintf(`
		UPDATE %s.onboarding_flows
		SET status = $1
		WHERE correlation_id = $2
	`, s.schema)
	_, err := s.pool.Exec(ctx, query, status, correlationID)
	return err
}

// CreateEvent inserts a new onboarding event linked to a flow.
func (s *PostgresStore) CreateEvent(ctx context.Context, event *models.OnboardingEvent) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.onboarding_events (id, flow_id, event_type, payload, status, retry_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, s.schema)

	_, err := s.pool.Exec(ctx, query,
		event.ID, event.FlowID, event.EventType,
		event.Payload, event.Status, event.RetryCount, event.CreatedAt,
	)
	return err
}

// UpdateEventStatus updates the status of an event and optionally sets an error message.
func (s *PostgresStore) UpdateEventStatus(ctx context.Context, eventID uuid.UUID, status string, errorMsg *string) error {
	query := fmt.Sprintf(`
		UPDATE %s.onboarding_events
		SET status = $1, error_message = $2, processed_at = $3
		WHERE id = $4
	`, s.schema)

	_, err := s.pool.Exec(ctx, query, status, errorMsg, time.Now(), eventID)
	return err
}

// GetEventsByFlowID retrieves all events for a given flow, ordered by creation time.
func (s *PostgresStore) GetEventsByFlowID(ctx context.Context, flowID uuid.UUID) ([]models.OnboardingEvent, error) {
	query := fmt.Sprintf(`
		SELECT id, flow_id, event_type, payload, status, error_message, retry_count, created_at, processed_at
		FROM %s.onboarding_events
		WHERE flow_id = $1
		ORDER BY created_at ASC
	`, s.schema)

	rows, err := s.pool.Query(ctx, query, flowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.OnboardingEvent
	for rows.Next() {
		var evt models.OnboardingEvent
		var payload json.RawMessage
		if err := rows.Scan(
			&evt.ID, &evt.FlowID, &evt.EventType,
			&payload, &evt.Status, &evt.ErrorMessage,
			&evt.RetryCount, &evt.CreatedAt, &evt.ProcessedAt,
		); err != nil {
			return nil, err
		}
		evt.Payload = payload
		events = append(events, evt)
	}
	return events, rows.Err()
}

// IncrementRetryCount increases the retry count of an event by 1.
func (s *PostgresStore) IncrementRetryCount(ctx context.Context, eventID uuid.UUID) error {
	query := fmt.Sprintf(`
		UPDATE %s.onboarding_events
		SET retry_count = retry_count + 1
		WHERE id = $1
	`, s.schema)

	_, err := s.pool.Exec(ctx, query, eventID)
	return err
}
