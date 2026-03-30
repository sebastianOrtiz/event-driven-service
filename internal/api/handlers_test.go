package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupTestRouter creates a Gin engine with the handlers wired to nil dependencies.
// This allows testing request parsing/validation without real store/publisher/redis.
func setupTestRouter() (*gin.Engine, *Handlers) {
	h := &Handlers{
		store:     nil,
		publisher: nil,
		redis:     nil,
	}

	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", h.HealthCheck)

	v1 := r.Group("/api/v1")
	v1.Use(apiKeyAuth("test-key"))
	{
		onboarding := v1.Group("/onboarding")
		{
			onboarding.GET("", h.ListFlows)
			onboarding.POST("/trigger", h.TriggerOnboarding)
			onboarding.GET("/:correlation_id", h.GetFlowStatus)
		}
	}

	return r, h
}

func TestHealthCheckPanicsWithNilRedis(t *testing.T) {
	// HealthCheck calls h.redis.Ping() which will panic with nil redis.
	// With gin.Recovery() middleware, this returns 500.
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// gin.Recovery catches the panic and returns 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 (nil redis panics), got %d", w.Code)
	}
}

func TestTriggerOnboardingMissingBody(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/trigger", nil)
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != "email and name are required" {
		t.Errorf("unexpected error message: %q", resp.Error)
	}
}

func TestTriggerOnboardingMissingEmail(t *testing.T) {
	router, _ := setupTestRouter()

	body := `{"name":"Test User"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/trigger", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestTriggerOnboardingMissingName(t *testing.T) {
	router, _ := setupTestRouter()

	body := `{"email":"test@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/trigger", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestTriggerOnboardingInvalidJSON(t *testing.T) {
	router, _ := setupTestRouter()

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/trigger", strings.NewReader(body))
	req.Header.Set("X-API-Key", "test-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetFlowStatusInvalidUUID(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/not-a-uuid", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != "invalid correlation_id format" {
		t.Errorf("unexpected error message: %q", resp.Error)
	}
}

func TestGetFlowStatusNilStorePanicsRecovered(t *testing.T) {
	// With a valid UUID but nil store, h.store.GetFlowByCorrelationID panics.
	// gin.Recovery returns 500.
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/550e8400-e29b-41d4-a716-446655440000", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 (nil store panics), got %d", w.Code)
	}
}

func TestListFlowsNilStorePanicsRecovered(t *testing.T) {
	// ListFlows with nil store panics, recovered by gin.Recovery.
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 (nil store panics), got %d", w.Code)
	}
}

func TestAPIKeyAuthMissingKey(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding", nil)
	// No X-API-Key header
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["error"] != "invalid or missing API key" {
		t.Errorf("unexpected error message: %q", resp["error"])
	}
}

func TestAPIKeyAuthWrongKey(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/onboarding", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	router, _ := setupTestRouter()
	router.Use(corsMiddleware())

	// OPTIONS preflight request
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/onboarding", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// CORS headers should be present
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin: got %q, want %q", got, "*")
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Errorf("Access-Control-Allow-Methods: got %q, want %q", got, "GET, POST, OPTIONS")
	}
}

func TestTriggerRequestBindingValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "empty body",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty JSON object",
			body:       "{}",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "only email",
			body:       `{"email":"a@b.com"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "only name",
			body:       `{"name":"Test"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	router, _ := setupTestRouter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body == "" {
				req = httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/trigger", nil)
			} else {
				req = httptest.NewRequest(http.MethodPost, "/api/v1/onboarding/trigger", strings.NewReader(tt.body))
			}
			req.Header.Set("X-API-Key", "test-key")
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status: got %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
