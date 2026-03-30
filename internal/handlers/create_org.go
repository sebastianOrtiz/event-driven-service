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

// CreateOrg handles email.verified events by simulating organization creation.
// After processing, it publishes an organization.created event.
type CreateOrg struct {
	store     *store.PostgresStore
	publisher *publisher.Publisher
}

// NewCreateOrg creates the create-org handler.
func NewCreateOrg(s *store.PostgresStore, p *publisher.Publisher) *CreateOrg {
	return &CreateOrg{store: s, publisher: p}
}

// Handle processes an email.verified event.
func (h *CreateOrg) Handle(ctx context.Context, payload events.EventPayload) error {
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
		if e.EventType == events.OrganizationCreated && e.Status == models.EventStatusCompleted {
			slog.Info("organization already created, skipping", "correlation_id", payload.CorrelationID)
			return nil
		}
	}

	// Record event as processing.
	eventID := uuid.New()
	payloadJSON, _ := json.Marshal(payload)
	evt := &models.OnboardingEvent{
		ID:        eventID,
		FlowID:    flow.ID,
		EventType: events.OrganizationCreated,
		Payload:   payloadJSON,
		Status:    models.EventStatusProcessing,
		CreatedAt: time.Now(),
	}
	if err := h.store.CreateEvent(ctx, evt); err != nil {
		return err
	}

	// Simulate organization creation.
	orgName := payload.OrgName
	if orgName == "" {
		orgName = payload.UserName + "'s Organization"
	}
	slog.Info("simulating organization creation",
		"correlation_id", payload.CorrelationID,
		"org_name", orgName,
	)
	time.Sleep(1 * time.Second)

	// Mark event completed.
	if err := h.store.UpdateEventStatus(ctx, eventID, models.EventStatusCompleted, nil); err != nil {
		return err
	}

	// Publish next event.
	payload.OrgName = orgName
	payload.Timestamp = time.Now().UTC().Format(time.RFC3339)
	return h.publisher.Publish(ctx, events.OrganizationCreated, payload)
}
