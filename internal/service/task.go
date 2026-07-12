package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/agentforge/agentforge/internal/repository"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// TaskPublisher dispatches a persisted task to an execution queue.
type TaskPublisher interface {
	Publish(ctx context.Context, taskID uuid.UUID) error
}

// SubmitTaskInput contains a new asynchronous task request.
type SubmitTaskInput struct {
	AgentID    uuid.UUID
	WorkflowID *uuid.UUID
	Priority   int
	Input      json.RawMessage
}

// TaskService manages durable task records before execution is dispatched to a queue.
type TaskService interface {
	Submit(ctx context.Context, input SubmitTaskInput) (*models.Task, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Task, error)
	Cancel(ctx context.Context, id uuid.UUID) error
}

type taskService struct {
	repository repository.TaskRepository
	publisher  TaskPublisher
}

// NewTaskService creates a task application service.
func NewTaskService(taskRepository repository.TaskRepository, publishers ...TaskPublisher) TaskService {
	var publisher TaskPublisher
	if len(publishers) > 0 {
		publisher = publishers[0]
	}
	return &taskService{repository: taskRepository, publisher: publisher}
}

func (s *taskService) Submit(ctx context.Context, input SubmitTaskInput) (*models.Task, error) {
	if input.AgentID == uuid.Nil {
		return nil, fmt.Errorf("%w: agent ID is required", ErrInvalidInput)
	}
	if input.Priority < 0 {
		return nil, fmt.Errorf("%w: priority must not be negative", ErrInvalidInput)
	}
	if !json.Valid(input.Input) {
		return nil, fmt.Errorf("%w: task input must be valid JSON", ErrInvalidInput)
	}
	task := &models.Task{
		ID:         uuid.New(),
		AgentID:    input.AgentID,
		WorkflowID: input.WorkflowID,
		Status:     models.TaskStatusPending,
		Priority:   input.Priority,
		Input:      datatypes.JSON(input.Input),
	}
	if err := s.repository.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("submit task service: %w", err)
	}
	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, task.ID); err != nil {
			return nil, fmt.Errorf("publish task service: %w", err)
		}
	}
	return task, nil
}

func (s *taskService) Get(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: task ID is required", ErrInvalidInput)
	}
	task, err := s.repository.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task service: %w", err)
	}
	return task, nil
}

func (s *taskService) Cancel(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: task ID is required", ErrInvalidInput)
	}
	for _, from := range []models.TaskStatus{models.TaskStatusPending, models.TaskStatusRunning} {
		err := s.repository.UpdateStatus(ctx, id, from, models.TaskStatusCancelled)
		if err == nil {
			return nil
		}
		if !errors.Is(err, repository.ErrConflict) {
			if errors.Is(err, repository.ErrNotFound) {
				return ErrNotFound
			}
			return fmt.Errorf("cancel task service: %w", err)
		}
	}
	return ErrConflict
}
