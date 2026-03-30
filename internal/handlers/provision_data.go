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

// ProvisionData handles organization.created events by simulating demo data provisioning.
// After processing, it publishes a demo_data.provisioned event.
type ProvisionData struct {
	store     *store.PostgresStore
	publisher *publisher.Publisher
}

// NewProvisionData creates the provision-data handler.
func NewProvisionData(s *store.PostgresStore, p *publisher.Publisher) *ProvisionData {
	return &ProvisionData{store: s, publisher: p}
}

// Handle processes an organization.created event.
func (h *ProvisionData) Handle(ctx context.Context, payload events.EventPayload) error {
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
		if e.EventType == events.DemoDataProvisioned && e.Status == models.EventStatusCompleted {
			slog.Info("demo data already provisioned, skipping", "correlation_id", payload.CorrelationID)
			return nil
		}
	}

	// Record event as processing.
	eventID := uuid.New()
	payloadJSON, _ := json.Marshal(payload)
	evt := &models.OnboardingEvent{
		ID:        eventID,
		FlowID:    flow.ID,
		EventType: events.DemoDataProvisioned,
		Payload:   payloadJSON,
		Status:    models.EventStatusProcessing,
		CreatedAt: time.Now(),
	}
	if err := h.store.CreateEvent(ctx, evt); err != nil {
		return err
	}

	// Simulate provisioning demo data (longest step).
	slog.Info("simulating demo data provisioning",
		"correlation_id", payload.CorrelationID,
		"org_name", payload.OrgName,
	)
	time.Sleep(2 * time.Second)

	// Mark event completed.
	if err := h.store.UpdateEventStatus(ctx, eventID, models.EventStatusCompleted, nil); err != nil {
		return err
	}

	// Publish next event.
	payload.Timestamp = time.Now().UTC().Format(time.RFC3339)
	return h.publisher.Publish(ctx, events.DemoDataProvisioned, payload)
}
