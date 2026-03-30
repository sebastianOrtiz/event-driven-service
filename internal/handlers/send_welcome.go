package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/sebasing/event-driven-service/internal/events"
	"github.com/sebasing/event-driven-service/internal/models"
	"github.com/sebasing/event-driven-service/internal/publisher"
	"github.com/sebasing/event-driven-service/internal/store"
)

// SendWelcome handles demo_data.provisioned events by simulating a welcome email.
// This is the final step — it publishes onboarding.completed and marks the flow as done.
type SendWelcome struct {
	store     *store.PostgresStore
	publisher *publisher.Publisher
}

// NewSendWelcome creates the send-welcome handler.
func NewSendWelcome(s *store.PostgresStore, p *publisher.Publisher) *SendWelcome {
	return &SendWelcome{store: s, publisher: p}
}

// Handle processes a demo_data.provisioned event.
func (h *SendWelcome) Handle(ctx context.Context, payload events.EventPayload) error {
	correlationID, err := uuid.Parse(payload.CorrelationID)
	if err != nil {
		return err
	}

	flow, err := h.store.GetFlowByCorrelationID(ctx, correlationID)
	if err != nil {
		return err
	}

	// Idempotency check.
	existingEvents, err := h.store.GetEventsByFlowID(ctx, flow.ID)
	if err != nil {
		return err
	}
	for _, e := range existingEvents {
		if e.EventType == events.OnboardingCompleted && e.Status == models.EventStatusCompleted {
			slog.Info("onboarding already completed, skipping", "correlation_id", payload.CorrelationID)
			return nil
		}
	}

	// Record event as processing.
	eventID := uuid.New()
	payloadJSON, _ := json.Marshal(payload)
	evt := &models.OnboardingEvent{
		ID:        eventID,
		FlowID:    flow.ID,
		EventType: events.OnboardingCompleted,
		Payload:   payloadJSON,
		Status:    models.EventStatusProcessing,
		CreatedAt: time.Now(),
	}
	if err := h.store.CreateEvent(ctx, evt); err != nil {
		return err
	}

	// Simulate sending welcome email.
	slog.Info("simulating welcome email",
		"correlation_id", payload.CorrelationID,
		"email", payload.UserEmail,
	)
	time.Sleep(500 * time.Millisecond)

	// Mark event completed.
	if err := h.store.UpdateEventStatus(ctx, eventID, models.EventStatusCompleted, nil); err != nil {
		return err
	}

	// Mark the entire flow as completed.
	if err := h.store.UpdateFlowStatus(ctx, correlationID, models.FlowStatusCompleted); err != nil {
		return err
	}

	// Publish the final event for observability.
	payload.Timestamp = time.Now().UTC().Format(time.RFC3339)
	return h.publisher.Publish(ctx, events.OnboardingCompleted, payload)
}
