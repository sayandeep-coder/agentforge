// Package api contains the HTTP delivery layer for AgentForge.
package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/agentforge/agentforge/internal/repository"
	"github.com/agentforge/agentforge/internal/service"
	"github.com/gin-gonic/gin"
)

// Response is the consistent API response envelope.
type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data"`
	Error   *Error `json:"error"`
}

// Error contains a stable machine-readable code and optional details.
type Error struct {
	Code    string `json:"code"`
	Details any    `json:"details,omitempty"`
}

func respond(c *gin.Context, status int, message string, data any) {
	c.JSON(status, Response{Success: true, Message: message, Data: data})
}

func respondError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	code := "INTERNAL_ERROR"
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		status, code = http.StatusUnprocessableEntity, "VALIDATION_ERROR"
	case errors.Is(err, service.ErrNotFound), errors.Is(err, repository.ErrNotFound):
		status, code = http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, service.ErrConflict), errors.Is(err, repository.ErrConflict):
		status, code = http.StatusConflict, "CONFLICT"
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		status, code = http.StatusRequestTimeout, "REQUEST_CANCELLED"
	}
	c.JSON(status, Response{
		Success: false,
		Message: err.Error(),
		Data:    nil,
		Error:   &Error{Code: code},
	})
}
