package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRouterHealthAndValidationEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter(Dependencies{})

	healthRequest := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRecorder := httptest.NewRecorder()
	router.ServeHTTP(healthRecorder, healthRequest)
	if healthRecorder.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", healthRecorder.Code)
	}
	if !strings.Contains(healthRecorder.Body.String(), `"success":true`) {
		t.Fatalf("health response = %s", healthRecorder.Body.String())
	}

	invalidRequest := httptest.NewRequest(http.MethodPost, "/v1/agents", strings.NewReader(`{}`))
	invalidRequest.Header.Set("Content-Type", "application/json")
	invalidRecorder := httptest.NewRecorder()
	router.ServeHTTP(invalidRecorder, invalidRequest)
	if invalidRecorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("validation status = %d, want 422", invalidRecorder.Code)
	}
	if !strings.Contains(invalidRecorder.Body.String(), `"code":"VALIDATION_ERROR"`) {
		t.Fatalf("validation response = %s", invalidRecorder.Body.String())
	}
}
