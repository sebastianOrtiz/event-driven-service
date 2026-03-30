// Package events defines event type constants and payload structures
// used throughout the onboarding pipeline.
package events

// Event type constants representing each step in the onboarding flow.
const (
	UserRegistered      = "user.registered"
	EmailVerified       = "email.verified"
	OrganizationCreated = "organization.created"
	DemoDataProvisioned = "demo_data.provisioned"
	OnboardingCompleted = "onboarding.completed"
)

// StreamName returns the Redis Stream name for a given event type.
func StreamName(eventType string) string {
	return "stream:" + eventType
}

// EventPayload is the canonical payload passed through the onboarding pipeline.
type EventPayload struct {
	CorrelationID string `json:"correlation_id"`
	UserEmail     string `json:"user_email"`
	UserName      string `json:"user_name"`
	OrgName       string `json:"org_name,omitempty"`
	Timestamp     string `json:"timestamp"`
}
