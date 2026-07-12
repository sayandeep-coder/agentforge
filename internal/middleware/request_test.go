package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestRequestIDUsesIncomingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/", func(c *gin.Context) {
		value, exists := c.Get("request_id")
		if !exists || value != "incoming-request" {
			t.Fatalf("request_id context value = %v, exists = %t", value, exists)
		}
		c.Status(204)
	})

	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set(requestIDHeader, "incoming-request")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if got := recorder.Header().Get(requestIDHeader); got != "incoming-request" {
		t.Fatalf("response request ID = %q", got)
	}
}

func TestAccessLogDoesNotChangeResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID(), AccessLog(zap.NewNop()))
	router.GET("/", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/", nil))
	if recorder.Code != 200 {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
}
