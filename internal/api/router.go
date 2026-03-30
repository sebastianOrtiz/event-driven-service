// Package api provides the HTTP API for the event-driven onboarding service.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/sebasing/event-driven-service/internal/publisher"
	"github.com/sebasing/event-driven-service/internal/store"
)

const apiKeyHeader = "X-API-Key"

// NewRouter builds and returns a Gin engine with all routes registered.
// The apiKey parameter is used to authenticate incoming requests.
func NewRouter(s *store.PostgresStore, pub *publisher.Publisher, rdb *redis.Client, apiKey string) *gin.Engine {
	h := &Handlers{
		store:     s,
		publisher: pub,
		redis:     rdb,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(corsMiddleware())

	// Health check is public — no API key required
	r.GET("/health", h.HealthCheck)

	// All /api/v1 endpoints require a valid API key
	v1 := r.Group("/api/v1")
	v1.Use(apiKeyAuth(apiKey))
	{
		onboarding := v1.Group("/onboarding")
		{
			onboarding.GET("", h.ListFlows)
			onboarding.POST("/trigger", h.TriggerOnboarding)
			onboarding.GET("/events", h.ListAllEvents)
			onboarding.GET("/:correlation_id", h.GetFlowStatus)
			onboarding.GET("/:correlation_id/events", h.GetFlowEvents)
		}
	}

	return r
}

// apiKeyAuth validates the X-API-Key header against the expected key.
func apiKeyAuth(expectedKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader(apiKeyHeader)
		if key == "" || key != expectedKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or missing API key",
			})
			return
		}
		c.Next()
	}
}

// corsMiddleware adds CORS headers for development.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
