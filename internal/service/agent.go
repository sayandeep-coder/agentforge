package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/agentforge/agentforge/internal/repository"
	"github.com/google/uuid"
)

const (
	defaultAgentTemperature = 0.7
	defaultAgentMaxTokens   = 4096
	maxAgentNameLength      = 100
)

// CreateAgentInput contains fields accepted when creating an agent.
type CreateAgentInput struct {
	Name         string
	Description  string
	SystemPrompt string
	Model        string
	Temperature  float64
	MaxTokens    int
}

// UpdateAgentInput contains optional mutable agent fields.
type UpdateAgentInput struct {
	Name         *string
	Description  *string
	SystemPrompt *string
	Model        *string
	Temperature  *float64
	MaxTokens    *int
	Status       *models.AgentStatus
}

// AgentService manages agent configuration and lifecycle rules.
type AgentService interface {
	Create(ctx context.Context, input CreateAgentInput) (*models.Agent, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	List(ctx context.Context, status models.AgentStatus, offset, limit int) ([]models.Agent, int64, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateAgentInput) (*models.Agent, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type agentService struct{ repository repository.AgentRepository }

// NewAgentService creates an agent application service.
func NewAgentService(agentRepository repository.AgentRepository) AgentService {
	return &agentService{repository: agentRepository}
}

func (s *agentService) Create(ctx context.Context, input CreateAgentInput) (*models.Agent, error) {
	if err := validateAgentInput(input); err != nil {
		return nil, err
	}
	temperature := input.Temperature
	if temperature == 0 {
		temperature = defaultAgentTemperature
	}
	maxTokens := input.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultAgentMaxTokens
	}
	agent := &models.Agent{
		ID:           uuid.New(),
		Name:         strings.TrimSpace(input.Name),
		Description:  strings.TrimSpace(input.Description),
		SystemPrompt: strings.TrimSpace(input.SystemPrompt),
		Model:        strings.TrimSpace(input.Model),
		Temperature:  temperature,
		MaxTokens:    maxTokens,
		Status:       models.AgentStatusActive,
	}
	if err := s.repository.Create(ctx, agent); err != nil {
		return nil, fmt.Errorf("create agent service: %w", err)
	}
	return agent, nil
}

func (s *agentService) Get(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: agent ID is required", ErrInvalidInput)
	}
	agent, err := s.repository.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get agent service: %w", err)
	}
	return agent, nil
}

func (s *agentService) List(ctx context.Context, status models.AgentStatus, offset, limit int) ([]models.Agent, int64, error) {
	if offset < 0 || limit < 1 || limit > 100 {
		return nil, 0, fmt.Errorf("%w: pagination must use offset >= 0 and limit between 1 and 100", ErrInvalidInput)
	}
	agents, total, err := s.repository.List(ctx, status, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list agents service: %w", err)
	}
	return agents, total, nil
}

func (s *agentService) Update(ctx context.Context, id uuid.UUID, input UpdateAgentInput) (*models.Agent, error) {
	agent, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := applyAgentUpdate(agent, input); err != nil {
		return nil, err
	}
	if err := s.repository.Update(ctx, agent); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update agent service: %w", err)
	}
	return agent, nil
}

func (s *agentService) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: agent ID is required", ErrInvalidInput)
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("delete agent service: %w", err)
	}
	return nil
}

func validateAgentInput(input CreateAgentInput) error {
	if name := strings.TrimSpace(input.Name); name == "" || len(name) > maxAgentNameLength {
		return fmt.Errorf("%w: agent name must be between 1 and %d characters", ErrInvalidInput, maxAgentNameLength)
	}
	if strings.TrimSpace(input.SystemPrompt) == "" {
		return fmt.Errorf("%w: system prompt is required", ErrInvalidInput)
	}
	if strings.TrimSpace(input.Model) == "" {
		return fmt.Errorf("%w: model is required", ErrInvalidInput)
	}
	if input.Temperature < 0 || input.Temperature > 1 {
		return fmt.Errorf("%w: temperature must be between 0 and 1", ErrInvalidInput)
	}
	if input.MaxTokens < 0 {
		return fmt.Errorf("%w: max tokens must not be negative", ErrInvalidInput)
	}
	return nil
}

func applyAgentUpdate(agent *models.Agent, input UpdateAgentInput) error {
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > maxAgentNameLength {
			return fmt.Errorf("%w: agent name must be between 1 and %d characters", ErrInvalidInput, maxAgentNameLength)
		}
		agent.Name = name
	}
	if input.Description != nil {
		agent.Description = strings.TrimSpace(*input.Description)
	}
	if input.SystemPrompt != nil {
		if strings.TrimSpace(*input.SystemPrompt) == "" {
			return fmt.Errorf("%w: system prompt is required", ErrInvalidInput)
		}
		agent.SystemPrompt = strings.TrimSpace(*input.SystemPrompt)
	}
	if input.Model != nil {
		if strings.TrimSpace(*input.Model) == "" {
			return fmt.Errorf("%w: model is required", ErrInvalidInput)
		}
		agent.Model = strings.TrimSpace(*input.Model)
	}
	if input.Temperature != nil {
		if *input.Temperature < 0 || *input.Temperature > 1 {
			return fmt.Errorf("%w: temperature must be between 0 and 1", ErrInvalidInput)
		}
		agent.Temperature = *input.Temperature
	}
	if input.MaxTokens != nil {
		if *input.MaxTokens < 1 {
			return fmt.Errorf("%w: max tokens must be positive", ErrInvalidInput)
		}
		agent.MaxTokens = *input.MaxTokens
	}
	if input.Status != nil {
		if *input.Status != models.AgentStatusActive && *input.Status != models.AgentStatusInactive {
			return fmt.Errorf("%w: unsupported agent status", ErrInvalidInput)
		}
		agent.Status = *input.Status
	}
	return nil
}
