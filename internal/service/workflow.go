package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/agentforge/agentforge/internal/repository"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const maxWorkflowNameLength = 100

// CreateWorkflowInput contains a workflow and its ordered steps.
type CreateWorkflowInput struct {
	AgentID     uuid.UUID
	Name        string
	Description string
	Steps       []WorkflowStepInput
}

// WorkflowStepInput contains the configuration for one workflow step.
type WorkflowStepInput struct {
	StepName   string
	StepType   string
	StepOrder  int
	ToolID     *uuid.UUID
	NextStepID *uuid.UUID
	Config     json.RawMessage
}

// UpdateWorkflowInput contains optional workflow metadata changes.
type UpdateWorkflowInput struct {
	Name        *string
	Description *string
	IsActive    *bool
}

// WorkflowService manages workflow definitions and step ordering.
type WorkflowService interface {
	Create(ctx context.Context, input CreateWorkflowInput) (*models.Workflow, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Workflow, error)
	ListByAgent(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]models.Workflow, int64, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateWorkflowInput) (*models.Workflow, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type workflowService struct{ repository repository.WorkflowRepository }

// NewWorkflowService creates a workflow application service.
func NewWorkflowService(workflowRepository repository.WorkflowRepository) WorkflowService {
	return &workflowService{repository: workflowRepository}
}

func (s *workflowService) Create(ctx context.Context, input CreateWorkflowInput) (*models.Workflow, error) {
	if err := validateWorkflowInput(input); err != nil {
		return nil, err
	}
	workflow := &models.Workflow{
		ID:          uuid.New(),
		AgentID:     input.AgentID,
		Name:        strings.TrimSpace(input.Name),
		Description: strings.TrimSpace(input.Description),
		Version:     1,
		IsActive:    true,
		Steps:       make([]models.WorkflowStep, 0, len(input.Steps)),
	}
	for _, inputStep := range input.Steps {
		config := inputStep.Config
		if len(config) == 0 {
			config = json.RawMessage(`{}`)
		}
		workflow.Steps = append(workflow.Steps, models.WorkflowStep{
			ID:         uuid.New(),
			WorkflowID: workflow.ID,
			StepName:   strings.TrimSpace(inputStep.StepName),
			StepType:   strings.TrimSpace(inputStep.StepType),
			StepOrder:  inputStep.StepOrder,
			ToolID:     inputStep.ToolID,
			NextStepID: inputStep.NextStepID,
			Config:     datatypes.JSON(config),
		})
	}
	if err := s.repository.Create(ctx, workflow); err != nil {
		return nil, fmt.Errorf("create workflow service: %w", err)
	}
	return workflow, nil
}

func (s *workflowService) Get(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: workflow ID is required", ErrInvalidInput)
	}
	workflow, err := s.repository.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get workflow service: %w", err)
	}
	return workflow, nil
}

func (s *workflowService) ListByAgent(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]models.Workflow, int64, error) {
	if agentID == uuid.Nil {
		return nil, 0, fmt.Errorf("%w: agent ID is required", ErrInvalidInput)
	}
	if offset < 0 || limit < 1 || limit > 100 {
		return nil, 0, fmt.Errorf("%w: pagination must use offset >= 0 and limit between 1 and 100", ErrInvalidInput)
	}
	workflows, total, err := s.repository.ListByAgent(ctx, agentID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("list workflows service: %w", err)
	}
	return workflows, total, nil
}

func (s *workflowService) Update(ctx context.Context, id uuid.UUID, input UpdateWorkflowInput) (*models.Workflow, error) {
	workflow, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > maxWorkflowNameLength {
			return nil, fmt.Errorf("%w: workflow name must be between 1 and %d characters", ErrInvalidInput, maxWorkflowNameLength)
		}
		workflow.Name = name
	}
	if input.Description != nil {
		workflow.Description = strings.TrimSpace(*input.Description)
	}
	if input.IsActive != nil {
		workflow.IsActive = *input.IsActive
	}
	if err := s.repository.Update(ctx, workflow); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update workflow service: %w", err)
	}
	return workflow, nil
}

func (s *workflowService) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: workflow ID is required", ErrInvalidInput)
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("delete workflow service: %w", err)
	}
	return nil
}

func validateWorkflowInput(input CreateWorkflowInput) error {
	if input.AgentID == uuid.Nil {
		return fmt.Errorf("%w: agent ID is required", ErrInvalidInput)
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > maxWorkflowNameLength {
		return fmt.Errorf("%w: workflow name must be between 1 and %d characters", ErrInvalidInput, maxWorkflowNameLength)
	}
	if len(input.Steps) == 0 {
		return fmt.Errorf("%w: workflow must contain at least one step", ErrInvalidInput)
	}
	orders := make(map[int]struct{}, len(input.Steps))
	for _, step := range input.Steps {
		if strings.TrimSpace(step.StepName) == "" || strings.TrimSpace(step.StepType) == "" || step.StepOrder < 1 {
			return fmt.Errorf("%w: every workflow step requires a name, type, and positive order", ErrInvalidInput)
		}
		if _, exists := orders[step.StepOrder]; exists {
			return fmt.Errorf("%w: workflow step orders must be unique", ErrInvalidInput)
		}
		orders[step.StepOrder] = struct{}{}
		if len(step.Config) > 0 && !json.Valid(step.Config) {
			return fmt.Errorf("%w: workflow step config must be valid JSON", ErrInvalidInput)
		}
	}
	ordered := make([]int, 0, len(orders))
	for order := range orders {
		ordered = append(ordered, order)
	}
	sort.Ints(ordered)
	for index, order := range ordered {
		if order != index+1 {
			return fmt.Errorf("%w: workflow step orders must be contiguous starting at 1", ErrInvalidInput)
		}
	}
	return nil
}
