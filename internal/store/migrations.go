// Package store provides PostgreSQL persistence for the event-driven service.
package store

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations creates the schema and tables if they don't exist.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, schema string) error {
	slog.Info("running database migrations", "schema", schema)

	queries := []string{
		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, schema),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.onboarding_flows (
			id UUID PRIMARY KEY,
			correlation_id UUID UNIQUE NOT NULL,
			user_email VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			completed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`, schema),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s.onboarding_events (
			id UUID PRIMARY KEY,
			flow_id UUID NOT NULL REFERENCES %s.onboarding_flows(id),
			event_type VARCHAR(100) NOT NULL,
			payload JSONB NOT NULL DEFAULT '{}',
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			error_message TEXT,
			retry_count INT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			processed_at TIMESTAMPTZ
		)`, schema, schema),

		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_events_flow_id ON %s.onboarding_events(flow_id)`, schema),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_flows_correlation_id ON %s.onboarding_flows(correlation_id)`, schema),
	}

	for _, q := range queries {
		if _, err := pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	slog.Info("database migrations completed successfully")
	return nil
}
