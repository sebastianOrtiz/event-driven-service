package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/sebasing/event-driven-service/internal/events"
	"github.com/sebasing/event-driven-service/internal/models"
	"github.com/sebasing/event-driven-service/internal/publisher"
	"github.com/sebasing/event-driven-service/internal/store"
)

// Handlers holds the dependencies for HTTP handlers.
type Handlers struct {
	store     *store.PostgresStore
	publisher *publisher.Publisher
	redis     *redis.Client
}

// TriggerRequest is the expected body for triggering onboarding.
type TriggerRequest struct {
	Email   string `json:"email" binding:"required"`
	Name    string `json:"name" binding:"required"`
	OrgName string `json:"orgName,omitempty"`
}

// TriggerResponse is returned after successfully starting onboarding.
type TriggerResponse struct {
	CorrelationID string `json:"correlationId"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// ErrorResponse is a generic error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// HealthCheck returns service health including Redis and PostgreSQL connectivity.
func (h *Handlers) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	redisStatus := "connected"
	if err := h.redis.Ping(ctx).Err(); err != nil {
		redisStatus = "disconnected"
	}

	pgStatus := "connected"

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"redis":    redisStatus,
		"postgres": pgStatus,
	})
}

// TriggerOnboarding creates a new onboarding flow and publishes the first event.
func (h *Handlers) TriggerOnboarding(c *gin.Context) {
	var req TriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email and name are required"})
		return
	}

	correlationID := uuid.New()
	now := time.Now()

	flow := &models.OnboardingFlow{
		ID:            uuid.New(),
		CorrelationID: correlationID,
		UserEmail:     req.Email,
		Status:        models.FlowStatusPending,
		StartedAt:     now,
		CreatedAt:     now,
	}

	ctx := c.Request.Context()
	if err := h.store.CreateFlow(ctx, flow); err != nil {
		slog.Error("failed to create flow", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create onboarding flow"})
		return
	}

	payload := events.EventPayload{
		CorrelationID: correlationID.String(),
		UserEmail:     req.Email,
		UserName:      req.Name,
		OrgName:       req.OrgName,
		Timestamp:     now.UTC().Format(time.RFC3339),
	}

	payloadJSON, _ := json.Marshal(payload)
	evt := &models.OnboardingEvent{
		ID:          uuid.New(),
		FlowID:      flow.ID,
		EventType:   events.UserRegistered,
		Payload:     payloadJSON,
		Status:      models.EventStatusCompleted,
		CreatedAt:   now,
		ProcessedAt: &now,
	}
	if err := h.store.CreateEvent(ctx, evt); err != nil {
		slog.Error("failed to record initial event", "error", err)
	}

	if err := h.publisher.Publish(ctx, events.UserRegistered, payload); err != nil {
		slog.Error("failed to publish user.registered event", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to trigger onboarding"})
		return
	}

	slog.Info("onboarding triggered",
		"correlation_id", correlationID.String(),
		"email", req.Email,
	)

	c.JSON(http.StatusAccepted, TriggerResponse{
		CorrelationID: correlationID.String(),
		Status:        "pending",
		Message:       "onboarding flow started",
	})
}

// GetFlowStatus returns the current status of an onboarding flow.
func (h *Handlers) GetFlowStatus(c *gin.Context) {
	correlationID, err := uuid.Parse(c.Param("correlation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid correlation_id format"})
		return
	}

	flow, err := h.store.GetFlowByCorrelationID(c.Request.Context(), correlationID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "flow not found"})
		return
	}

	c.JSON(http.StatusOK, flow)
}

// GetFlowEvents returns all events for a given onboarding flow.
func (h *Handlers) GetFlowEvents(c *gin.Context) {
	correlationID, err := uuid.Parse(c.Param("correlation_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid correlation_id format"})
		return
	}

	flow, err := h.store.GetFlowByCorrelationID(c.Request.Context(), correlationID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "flow not found"})
		return
	}

	evts, err := h.store.GetEventsByFlowID(c.Request.Context(), flow.ID)
	if err != nil {
		slog.Error("failed to fetch events", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to fetch events"})
		return
	}

	if evts == nil {
		evts = []models.OnboardingEvent{}
	}

	c.JSON(http.StatusOK, gin.H{
		"correlationId": correlationID.String(),
		"events":        evts,
	})
}

// ListFlows returns all onboarding flows ordered by creation time.
func (h *Handlers) ListFlows(c *gin.Context) {
	flows, err := h.store.ListFlows(c.Request.Context(), 50)
	if err != nil {
		slog.Error("failed to list flows", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to list flows"})
		return
	}
	if flows == nil {
		flows = []models.OnboardingFlow{}
	}
	c.JSON(http.StatusOK, gin.H{"flows": flows})
}

// ListAllEvents returns all events across all flows.
func (h *Handlers) ListAllEvents(c *gin.Context) {
	evts, err := h.store.ListAllEvents(c.Request.Context(), 100)
	if err != nil {
		slog.Error("failed to list events", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to list events"})
		return
	}
	if evts == nil {
		evts = []models.OnboardingEvent{}
	}
	c.JSON(http.StatusOK, gin.H{"events": evts})
}
