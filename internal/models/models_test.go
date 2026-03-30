package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFlowStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"Pending", FlowStatusPending, "pending"},
		{"InProgress", FlowStatusInProgress, "in_progress"},
		{"Completed", FlowStatusCompleted, "completed"},
		{"Failed", FlowStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.got)
			}
		})
	}
}

func TestEventStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"Pending", EventStatusPending, "pending"},
		{"Processing", EventStatusProcessing, "processing"},
		{"Completed", EventStatusCompleted, "completed"},
		{"Failed", EventStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.got)
			}
		})
	}
}

func TestOnboardingFlowCreation(t *testing.T) {
	id := uuid.New()
	corrID := uuid.New()
	now := time.Now()

	flow := OnboardingFlow{
		ID:            id,
		CorrelationID: corrID,
		UserEmail:     "test@example.com",
		Status:        FlowStatusPending,
		StartedAt:     now,
		CreatedAt:     now,
	}

	if flow.ID != id {
		t.Errorf("ID: got %v, want %v", flow.ID, id)
	}
	if flow.CorrelationID != corrID {
		t.Errorf("CorrelationID: got %v, want %v", flow.CorrelationID, corrID)
	}
	if flow.UserEmail != "test@example.com" {
		t.Errorf("UserEmail: got %q, want %q", flow.UserEmail, "test@example.com")
	}
	if flow.Status != FlowStatusPending {
		t.Errorf("Status: got %q, want %q", flow.Status, FlowStatusPending)
	}
	if flow.CompletedAt != nil {
		t.Error("CompletedAt should be nil for a new flow")
	}
}

func TestOnboardingFlowJSONSerialization(t *testing.T) {
	id := uuid.New()
	corrID := uuid.New()
	now := time.Now().Truncate(time.Second)

	flow := OnboardingFlow{
		ID:            id,
		CorrelationID: corrID,
		UserEmail:     "user@test.com",
		Status:        FlowStatusInProgress,
		StartedAt:     now,
		CreatedAt:     now,
	}

	data, err := json.Marshal(flow)
	if err != nil {
		t.Fatalf("failed to marshal OnboardingFlow: %v", err)
	}

	var decoded OnboardingFlow
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal OnboardingFlow: %v", err)
	}

	if decoded.ID != flow.ID {
		t.Errorf("ID mismatch: got %v, want %v", decoded.ID, flow.ID)
	}
	if decoded.Status != flow.Status {
		t.Errorf("Status mismatch: got %q, want %q", decoded.Status, flow.Status)
	}
	if decoded.UserEmail != flow.UserEmail {
		t.Errorf("UserEmail mismatch: got %q, want %q", decoded.UserEmail, flow.UserEmail)
	}
}

func TestOnboardingEventCreation(t *testing.T) {
	id := uuid.New()
	flowID := uuid.New()
	now := time.Now()
	payload := json.RawMessage(`{"key":"value"}`)

	evt := OnboardingEvent{
		ID:         id,
		FlowID:     flowID,
		EventType:  "user.registered",
		Payload:    payload,
		Status:     EventStatusPending,
		RetryCount: 0,
		CreatedAt:  now,
	}

	if evt.ID != id {
		t.Errorf("ID: got %v, want %v", evt.ID, id)
	}
	if evt.FlowID != flowID {
		t.Errorf("FlowID: got %v, want %v", evt.FlowID, flowID)
	}
	if evt.EventType != "user.registered" {
		t.Errorf("EventType: got %q, want %q", evt.EventType, "user.registered")
	}
	if evt.Status != EventStatusPending {
		t.Errorf("Status: got %q, want %q", evt.Status, EventStatusPending)
	}
	if evt.RetryCount != 0 {
		t.Errorf("RetryCount: got %d, want 0", evt.RetryCount)
	}
	if evt.ErrorMessage != nil {
		t.Error("ErrorMessage should be nil for a new event")
	}
	if evt.ProcessedAt != nil {
		t.Error("ProcessedAt should be nil for a new event")
	}
}

func TestOnboardingEventWithErrorMessage(t *testing.T) {
	errMsg := "connection timeout"
	evt := OnboardingEvent{
		ID:           uuid.New(),
		FlowID:       uuid.New(),
		EventType:    "email.verified",
		Payload:      json.RawMessage(`{}`),
		Status:       EventStatusFailed,
		ErrorMessage: &errMsg,
		RetryCount:   3,
		CreatedAt:    time.Now(),
	}

	if evt.ErrorMessage == nil {
		t.Fatal("ErrorMessage should not be nil")
	}
	if *evt.ErrorMessage != errMsg {
		t.Errorf("ErrorMessage: got %q, want %q", *evt.ErrorMessage, errMsg)
	}
	if evt.RetryCount != 3 {
		t.Errorf("RetryCount: got %d, want 3", evt.RetryCount)
	}
}
