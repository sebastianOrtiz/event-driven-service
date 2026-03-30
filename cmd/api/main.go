// Command api starts the HTTP API server for the event-driven onboarding service.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/sebasing/event-driven-service/internal/api"
	"github.com/sebasing/event-driven-service/internal/config"
	"github.com/sebasing/event-driven-service/internal/publisher"
	"github.com/sebasing/event-driven-service/internal/store"
)

func main() {
	// Structured JSON logging.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg := config.Load()
	ctx := context.Background()

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

	// Build dependencies.
	pgStore := store.NewPostgresStore(pool, cfg.DBSchema)
	pub := publisher.New(rdb)

	// Create HTTP server.
	router := api.NewRouter(pgStore, pub, rdb, cfg.APIKey)
	slog.Info("API key authentication enabled")
	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		slog.Info("shutting down API server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown error", "error", err)
		}
	}()

	slog.Info("API server starting", "port", cfg.HTTPPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}

	slog.Info("API server stopped")
}
