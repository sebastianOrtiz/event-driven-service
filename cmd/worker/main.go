// Command worker starts all onboarding pipeline workers in a single process.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/sebasing/event-driven-service/internal/config"
	"github.com/sebasing/event-driven-service/internal/consumer"
	"github.com/sebasing/event-driven-service/internal/events"
	"github.com/sebasing/event-driven-service/internal/handlers"
	"github.com/sebasing/event-driven-service/internal/publisher"
	"github.com/sebasing/event-driven-service/internal/store"
)

func main() {
	// Structured JSON logging.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to PostgreSQL.
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to PostgreSQL", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("failed to ping PostgreSQL", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to PostgreSQL")

	// Run migrations.
	if err := store.RunMigrations(ctx, pool, cfg.DBSchema); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Connect to Redis.
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisURL,
		Password: cfg.RedisPassword,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	slog.Info("connected to Redis")

	// Build shared dependencies.
	pgStore := store.NewPostgresStore(pool, cfg.DBSchema)
	pub := publisher.New(rdb)

	// Create handlers.
	verifyEmail := handlers.NewVerifyEmail(pgStore, pub)
	createOrg := handlers.NewCreateOrg(pgStore, pub)
	provisionData := handlers.NewProvisionData(pgStore, pub)
	sendWelcome := handlers.NewSendWelcome(pgStore, pub)

	// Define the worker pipeline: each consumer listens to a specific stream.
	consumers := []*consumer.Consumer{
		consumer.New(rdb, cfg.ConsumerGroup, "verify-email", events.UserRegistered, verifyEmail.Handle, cfg.MaxRetries, cfg.RetryBackoff),
		consumer.New(rdb, cfg.ConsumerGroup, "create-org", events.EmailVerified, createOrg.Handle, cfg.MaxRetries, cfg.RetryBackoff),
		consumer.New(rdb, cfg.ConsumerGroup, "provision-data", events.OrganizationCreated, provisionData.Handle, cfg.MaxRetries, cfg.RetryBackoff),
		consumer.New(rdb, cfg.ConsumerGroup, "send-welcome", events.DemoDataProvisioned, sendWelcome.Handle, cfg.MaxRetries, cfg.RetryBackoff),
	}

	// Ensure consumer groups exist for all streams.
	for _, c := range consumers {
		if err := c.EnsureGroup(ctx); err != nil {
			slog.Error("failed to ensure consumer group", "error", err)
			os.Exit(1)
		}
	}

	// Start all workers as goroutines.
	for _, c := range consumers {
		go c.Run(ctx)
	}

	slog.Info("all workers started", "count", len(consumers))

	// Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh

	slog.Info("received shutdown signal", "signal", sig.String())
	cancel() // Cancel context to stop all consumers.
	slog.Info("all workers stopped")
}
