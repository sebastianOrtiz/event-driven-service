// Package config provides configuration loading from environment variables.
package config

import (
	"os"
	"strconv"
)

// Config holds all configuration values for the service.
type Config struct {
	RedisURL      string
	DatabaseURL   string
	HTTPPort      string
	DBSchema      string
	MaxRetries    int
	RetryBackoff  int // milliseconds
	ConsumerGroup string
	APIKey        string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://sebasing:devpassword@localhost:5432/sebasing_dev?sslmode=disable"),
		HTTPPort:      getEnv("HTTP_PORT", "8081"),
		DBSchema:      getEnv("DB_SCHEMA", "events"),
		MaxRetries:    getEnvInt("MAX_RETRIES", 3),
		RetryBackoff:  getEnvInt("RETRY_BACKOFF_MS", 1000),
		ConsumerGroup: getEnv("CONSUMER_GROUP", "onboarding-workers"),
		APIKey:        getEnv("API_KEY", "nexus-events-dev-key-2026"),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}
