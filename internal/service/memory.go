package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/agentforge/agentforge/internal/repository"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// MemoryService manages durable JSON memory for agents.
type MemoryService interface {
	Get(ctx context.Context, agentID uuid.UUID, key string) (*models.AgentMemory, error)
	Upsert(ctx context.Context, agentID uuid.UUID, key string, value json.RawMessage) (*models.AgentMemory, error)
	Delete(ctx context.Context, agentID uuid.UUID, key string) error
}

type memoryService struct{ repository repository.MemoryRepository }

// NewMemoryService creates a memory application service.
func NewMemoryService(memoryRepository repository.MemoryRepository) MemoryService {
	return &memoryService{repository: memoryRepository}
}

func (s *memoryService) Get(ctx context.Context, agentID uuid.UUID, key string) (*models.AgentMemory, error) {
	if err := validateMemoryReference(agentID, key); err != nil {
		return nil, err
	}
	memory, err := s.repository.Get(ctx, agentID, key)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get memory service: %w", err)
	}
	return memory, nil
}

func (s *memoryService) Upsert(ctx context.Context, agentID uuid.UUID, key string, value json.RawMessage) (*models.AgentMemory, error) {
	if err := validateMemoryReference(agentID, key); err != nil {
		return nil, err
	}
	if !json.Valid(value) {
		return nil, fmt.Errorf("%w: memory value must be valid JSON", ErrInvalidInput)
	}
	memory := &models.AgentMemory{
		ID:          uuid.New(),
		AgentID:     agentID,
		MemoryKey:   strings.TrimSpace(key),
		MemoryValue: datatypes.JSON(value),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := s.repository.Upsert(ctx, memory); err != nil {
		return nil, fmt.Errorf("upsert memory service: %w", err)
	}
	return memory, nil
}

func (s *memoryService) Delete(ctx context.Context, agentID uuid.UUID, key string) error {
	if err := validateMemoryReference(agentID, key); err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, agentID, key); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("delete memory service: %w", err)
	}
	return nil
}

func validateMemoryReference(agentID uuid.UUID, key string) error {
	if agentID == uuid.Nil {
		return fmt.Errorf("%w: agent ID is required", ErrInvalidInput)
	}
	if key = strings.TrimSpace(key); key == "" || len(key) > 255 {
		return fmt.Errorf("%w: memory key must be between 1 and 255 characters", ErrInvalidInput)
	}
	return nil
}
