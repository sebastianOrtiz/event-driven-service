// Package consumer provides Redis Streams consumer group functionality.
package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/sebasing/event-driven-service/internal/events"
)

// HandlerFunc processes an event payload. Returns an error if processing fails.
type HandlerFunc func(ctx context.Context, payload events.EventPayload) error

// Consumer reads from a Redis Stream using consumer groups.
type Consumer struct {
	client        *redis.Client
	group         string
	consumerName  string
	stream        string
	handler       HandlerFunc
	maxRetries    int
	retryBackoff  int // base backoff in milliseconds
}

// New creates a new Consumer for the given stream and event type.
func New(
	client *redis.Client,
	group string,
	consumerName string,
	eventType string,
	handler HandlerFunc,
	maxRetries int,
	retryBackoff int,
) *Consumer {
	return &Consumer{
		client:       client,
		group:        group,
		consumerName: consumerName,
		stream:       events.StreamName(eventType),
		handler:      handler,
		maxRetries:   maxRetries,
		retryBackoff: retryBackoff,
	}
}

// EnsureGroup creates the consumer group if it does not already exist.
// It uses MKSTREAM to create the stream automatically.
func (c *Consumer) EnsureGroup(ctx context.Context) error {
	err := c.client.XGroupCreateMkStream(ctx, c.stream, c.group, "0").Err()
	if err != nil {
		// Ignore "BUSYGROUP" error — group already exists.
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			return nil
		}
		return fmt.Errorf("failed to create consumer group %s on %s: %w", c.group, c.stream, err)
	}
	slog.Info("consumer group created", "group", c.group, "stream", c.stream)
	return nil
}

// Run starts consuming messages in a loop until the context is cancelled.
func (c *Consumer) Run(ctx context.Context) {
	slog.Info("consumer started", "stream", c.stream, "group", c.group, "consumer", c.consumerName)

	for {
		select {
		case <-ctx.Done():
			slog.Info("consumer stopping", "stream", c.stream)
			return
		default:
		}

		messages, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.group,
			Consumer: c.consumerName,
			Streams:  []string{c.stream, ">"},
			Count:    1,
			Block:    5 * time.Second,
		}).Result()

		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			// redis.Nil means no new messages — not an error.
			if errors.Is(err, redis.Nil) {
				continue
			}
			slog.Error("failed to read from stream", "stream", c.stream, "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range messages {
			for _, msg := range stream.Messages {
				c.processMessage(ctx, msg)
			}
		}
	}
}

// processMessage deserializes and handles a single message, with retries.
func (c *Consumer) processMessage(ctx context.Context, msg redis.XMessage) {
	data, ok := msg.Values["data"].(string)
	if !ok {
		slog.Error("invalid message format", "message_id", msg.ID, "stream", c.stream)
		c.ack(ctx, msg.ID)
		return
	}

	var payload events.EventPayload
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		slog.Error("failed to unmarshal payload", "message_id", msg.ID, "error", err)
		c.ack(ctx, msg.ID)
		return
	}

	slog.Info("processing event",
		"stream", c.stream,
		"message_id", msg.ID,
		"correlation_id", payload.CorrelationID,
	)

	// Attempt processing with retries and exponential backoff.
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(float64(c.retryBackoff)*math.Pow(2, float64(attempt-1))) * time.Millisecond
			slog.Warn("retrying event processing",
				"stream", c.stream,
				"attempt", attempt,
				"backoff_ms", backoff.Milliseconds(),
				"correlation_id", payload.CorrelationID,
			)
			time.Sleep(backoff)
		}

		if err := c.handler(ctx, payload); err != nil {
			lastErr = err
			slog.Error("handler failed",
				"stream", c.stream,
				"attempt", attempt,
				"error", err,
				"correlation_id", payload.CorrelationID,
			)
			continue
		}

		// Success — acknowledge the message.
		c.ack(ctx, msg.ID)
		slog.Info("event processed successfully",
			"stream", c.stream,
			"message_id", msg.ID,
			"correlation_id", payload.CorrelationID,
		)
		return
	}

	// All retries exhausted.
	slog.Error("event processing failed after all retries",
		"stream", c.stream,
		"message_id", msg.ID,
		"correlation_id", payload.CorrelationID,
		"error", lastErr,
	)
	c.ack(ctx, msg.ID) // ACK to avoid reprocessing forever; event is recorded as failed.
}

// ack acknowledges a message in the consumer group.
func (c *Consumer) ack(ctx context.Context, messageID string) {
	if err := c.client.XAck(ctx, c.stream, c.group, messageID).Err(); err != nil {
		slog.Error("failed to ACK message", "stream", c.stream, "message_id", messageID, "error", err)
	}
}
