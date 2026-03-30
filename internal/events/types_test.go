package events

import (
	"encoding/json"
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"UserRegistered", UserRegistered, "user.registered"},
		{"EmailVerified", EmailVerified, "email.verified"},
		{"OrganizationCreated", OrganizationCreated, "organization.created"},
		{"DemoDataProvisioned", DemoDataProvisioned, "demo_data.provisioned"},
		{"OnboardingCompleted", OnboardingCompleted, "onboarding.completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.got)
			}
		})
	}
}

func TestStreamName(t *testing.T) {
	tests := []struct {
		eventType string
		expected  string
	}{
		{UserRegistered, "stream:user.registered"},
		{EmailVerified, "stream:email.verified"},
		{OrganizationCreated, "stream:organization.created"},
		{DemoDataProvisioned, "stream:demo_data.provisioned"},
		{OnboardingCompleted, "stream:onboarding.completed"},
		{"custom.event", "stream:custom.event"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			got := StreamName(tt.eventType)
			if got != tt.expected {
				t.Errorf("StreamName(%q) = %q, want %q", tt.eventType, got, tt.expected)
			}
		})
	}
}

func TestEventPayloadJSONMarshal(t *testing.T) {
	payload := EventPayload{
		CorrelationID: "abc-123",
		UserEmail:     "test@example.com",
		UserName:      "Test User",
		OrgName:       "Test Org",
		Timestamp:     "2026-03-30T12:00:00Z",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal EventPayload: %v", err)
	}

	var decoded EventPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal EventPayload: %v", err)
	}

	if decoded.CorrelationID != payload.CorrelationID {
		t.Errorf("CorrelationID: got %q, want %q", decoded.CorrelationID, payload.CorrelationID)
	}
	if decoded.UserEmail != payload.UserEmail {
		t.Errorf("UserEmail: got %q, want %q", decoded.UserEmail, payload.UserEmail)
	}
	if decoded.UserName != payload.UserName {
		t.Errorf("UserName: got %q, want %q", decoded.UserName, payload.UserName)
	}
	if decoded.OrgName != payload.OrgName {
		t.Errorf("OrgName: got %q, want %q", decoded.OrgName, payload.OrgName)
	}
	if decoded.Timestamp != payload.Timestamp {
		t.Errorf("Timestamp: got %q, want %q", decoded.Timestamp, payload.Timestamp)
	}
}

func TestEventPayloadOmitEmptyOrgName(t *testing.T) {
	payload := EventPayload{
		CorrelationID: "abc-123",
		UserEmail:     "test@example.com",
		UserName:      "Test User",
		Timestamp:     "2026-03-30T12:00:00Z",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal EventPayload: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	if _, ok := raw["org_name"]; ok {
		t.Error("expected org_name to be omitted when empty, but it was present")
	}
}

func TestEventPayloadUnmarshalFromJSON(t *testing.T) {
	jsonStr := `{
		"correlation_id": "xyz-789",
		"user_email": "user@test.com",
		"user_name": "Jane Doe",
		"org_name": "Acme Corp",
		"timestamp": "2026-01-15T10:30:00Z"
	}`

	var payload EventPayload
	if err := json.Unmarshal([]byte(jsonStr), &payload); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if payload.CorrelationID != "xyz-789" {
		t.Errorf("CorrelationID: got %q, want %q", payload.CorrelationID, "xyz-789")
	}
	if payload.UserEmail != "user@test.com" {
		t.Errorf("UserEmail: got %q, want %q", payload.UserEmail, "user@test.com")
	}
	if payload.OrgName != "Acme Corp" {
		t.Errorf("OrgName: got %q, want %q", payload.OrgName, "Acme Corp")
	}
}
