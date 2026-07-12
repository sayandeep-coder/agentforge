package api

import (
	"fmt"
	"strconv"

	"github.com/agentforge/agentforge/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func bindJSON(c *gin.Context, value any) error {
	if err := c.ShouldBindJSON(value); err != nil {
		return fmt.Errorf("%w: %v", service.ErrInvalidInput, err)
	}
	return nil
}

func pathUUID(c *gin.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid %s", service.ErrInvalidInput, name)
	}
	return id, nil
}

func parseRequestUUID(value, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid %s", service.ErrInvalidInput, name)
	}
	return id, nil
}

func optionalUUID(value *string, name string) (*uuid.UUID, error) {
	if value == nil || *value == "" {
		return nil, nil
	}
	id, err := parseRequestUUID(*value, name)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func pagination(c *gin.Context) (int, int, error) {
	offset, err := queryInt(c, "offset", 0)
	if err != nil {
		return 0, 0, err
	}
	limit, err := queryInt(c, "limit", 20)
	if err != nil {
		return 0, 0, err
	}
	return offset, limit, nil
}

func queryInt(c *gin.Context, key string, fallback int) (int, error) {
	value := c.Query(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid %s: must be an integer", service.ErrInvalidInput, key)
	}
	return parsed, nil
}
