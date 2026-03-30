// Package handlers contains the worker handlers for each onboarding step.
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

// VerifyEmail handles user.registered events by simulating email verification.
// After processing, it publishes an email.verified event.
type VerifyEmail struct {
	store     *store.PostgresStore
	publisher *publisher.Publisher
}

// NewVerifyEmail creates the verify-email handler.
func NewVerifyEmail(s *store.PostgresStore, p *publisher.Publisher) *VerifyEmail {
	return &VerifyEmail{store: s, publisher: p}
}

// Handle processes a user.registered event.
func (h *VerifyEmail) Handle(ctx context.Context, payload events.EventPayload) error {
	correlationID, err := uuid.Parse(payload.CorrelationID)
	if err != nil {
		return err
	}

	// Idempotency: check if already processed by looking at existing events.
	flow, err := h.store.GetFlowByCorrelationID(ctx, correlationID)
	if err != nil {
		return err
	}

	existingEvents, err := h.store.GetEventsByFlowID(ctx, flow.ID)
	if err != nil {
		return err
	}
	for _, e := range existingEvents {
		if e.EventType == events.EmailVerified && e.Status == models.EventStatusCompleted {
			slog.Info("email already verified, skipping", "correlation_id", payload.CorrelationID)
			return nil
		}
	}

	// Update flow to in_progress.
	if err := h.store.UpdateFlowStatus(ctx, correlationID, models.FlowStatusInProgress); err != nil {
		return err
	}

	// Record the event as processing.
	eventID := uuid.New()
	payloadJSON, _ := json.Marshal(payload)
	evt := &models.OnboardingEvent{
		ID:        eventID,
		FlowID:    flow.ID,
		EventType: events.EmailVerified,
		Payload:   payloadJSON,
		Status:    models.EventStatusProcessing,
		CreatedAt: time.Now(),
	}
	if err := h.store.CreateEvent(ctx, evt); err != nil {
		return err
	}

	// Simulate email verification work.
	slog.Info("simulating email verification", "correlation_id", payload.CorrelationID, "email", payload.UserEmail)
	time.Sleep(800 * time.Millisecond)

	// Mark event as completed.
	if err := h.store.UpdateEventStatus(ctx, eventID, models.EventStatusCompleted, nil); err != nil {
		return err
	}

	// Publish next event in the chain.
	payload.Timestamp = time.Now().UTC().Format(time.RFC3339)
	return h.publisher.Publish(ctx, events.EmailVerified, payload)
}
