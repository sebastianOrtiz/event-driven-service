// Package publisher provides Redis Streams event publishing.
package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"

	"github.com/sebasing/event-driven-service/internal/events"
)

// Publisher sends events to Redis Streams.
type Publisher struct {
	client *redis.Client
}

// New creates a new Publisher with the given Redis client.
func New(client *redis.Client) *Publisher {
	return &Publisher{client: client}
}

// Publish sends an event payload to the appropriate Redis Stream.
// The stream name is derived from the event type: "stream:<event_type>".
func (p *Publisher) Publish(ctx context.Context, eventType string, payload events.EventPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	stream := events.StreamName(eventType)

	result := p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"data": string(data),
		},
	})

	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to publish event to stream %s: %w", stream, err)
	}

	slog.Info("event published",
		"stream", stream,
		"event_type", eventType,
		"correlation_id", payload.CorrelationID,
		"message_id", result.Val(),
	)

	return nil
}
