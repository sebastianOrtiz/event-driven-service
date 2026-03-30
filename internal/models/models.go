// Package models defines the domain models persisted in PostgreSQL.
package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OnboardingFlow represents a single user's onboarding process.
type OnboardingFlow struct {
	ID            uuid.UUID  `json:"id"`
	CorrelationID uuid.UUID  `json:"correlationId"`
	UserEmail     string     `json:"userEmail"`
	Status        string     `json:"status"` // pending, in_progress, completed, failed
	StartedAt     time.Time  `json:"startedAt"`
	CompletedAt   *time.Time `json:"completedAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// OnboardingEvent represents a single event within an onboarding flow.
type OnboardingEvent struct {
	ID           uuid.UUID        `json:"id"`
	FlowID       uuid.UUID        `json:"flowId"`
	EventType    string           `json:"eventType"`
	Payload      json.RawMessage  `json:"payload"`
	Status       string           `json:"status"` // pending, processing, completed, failed
	ErrorMessage *string          `json:"errorMessage,omitempty"`
	RetryCount   int              `json:"retryCount"`
	CreatedAt    time.Time        `json:"createdAt"`
	ProcessedAt  *time.Time       `json:"processedAt,omitempty"`
}

// Flow statuses.
const (
	FlowStatusPending    = "pending"
	FlowStatusInProgress = "in_progress"
	FlowStatusCompleted  = "completed"
	FlowStatusFailed     = "failed"
)

// Event statuses.
const (
	EventStatusPending    = "pending"
	EventStatusProcessing = "processing"
	EventStatusCompleted  = "completed"
	EventStatusFailed     = "failed"
)
