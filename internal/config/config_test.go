package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg := Load()

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"RedisURL", cfg.RedisURL, "redis:6379"},
		{"HTTPPort", cfg.HTTPPort, "8081"},
		{"DBSchema", cfg.DBSchema, "events"},
		{"ConsumerGroup", cfg.ConsumerGroup, "onboarding-workers"},
		{"APIKey", cfg.APIKey, "nexus-events-dev-key-2026"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.got)
			}
		})
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries: expected 3, got %d", cfg.MaxRetries)
	}
	if cfg.RetryBackoff != 1000 {
		t.Errorf("RetryBackoff: expected 1000, got %d", cfg.RetryBackoff)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("REDIS_URL", "localhost:6380")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("DB_SCHEMA", "test_events")
	t.Setenv("MAX_RETRIES", "5")
	t.Setenv("RETRY_BACKOFF_MS", "2000")
	t.Setenv("CONSUMER_GROUP", "test-workers")
	t.Setenv("API_KEY", "test-api-key")
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/testdb")

	cfg := Load()

	if cfg.RedisURL != "localhost:6380" {
		t.Errorf("RedisURL: expected %q, got %q", "localhost:6380", cfg.RedisURL)
	}
	if cfg.HTTPPort != "9090" {
		t.Errorf("HTTPPort: expected %q, got %q", "9090", cfg.HTTPPort)
	}
	if cfg.DBSchema != "test_events" {
		t.Errorf("DBSchema: expected %q, got %q", "test_events", cfg.DBSchema)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries: expected 5, got %d", cfg.MaxRetries)
	}
	if cfg.RetryBackoff != 2000 {
		t.Errorf("RetryBackoff: expected 2000, got %d", cfg.RetryBackoff)
	}
	if cfg.ConsumerGroup != "test-workers" {
		t.Errorf("ConsumerGroup: expected %q, got %q", "test-workers", cfg.ConsumerGroup)
	}
	if cfg.APIKey != "test-api-key" {
		t.Errorf("APIKey: expected %q, got %q", "test-api-key", cfg.APIKey)
	}
	if cfg.DatabaseURL != "postgres://test:test@localhost:5432/testdb" {
		t.Errorf("DatabaseURL: expected postgres://test:test@localhost:5432/testdb, got %q", cfg.DatabaseURL)
	}
}

func TestLoadInvalidIntFallsBackToDefault(t *testing.T) {
	t.Setenv("MAX_RETRIES", "not-a-number")
	t.Setenv("RETRY_BACKOFF_MS", "invalid")

	cfg := Load()

	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries: expected default 3 for invalid env, got %d", cfg.MaxRetries)
	}
	if cfg.RetryBackoff != 1000 {
		t.Errorf("RetryBackoff: expected default 1000 for invalid env, got %d", cfg.RetryBackoff)
	}
}
