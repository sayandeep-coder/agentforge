package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/agentforge/agentforge/internal/repository"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// CreateScheduleInput contains a cron schedule definition.
type CreateScheduleInput struct {
	AgentID        uuid.UUID
	WorkflowID     *uuid.UUID
	CronExpression string
	Enabled        bool
}

// UpdateScheduleInput contains optional schedule changes.
type UpdateScheduleInput struct {
	WorkflowID     *uuid.UUID
	CronExpression *string
	Enabled        *bool
}

// ScheduleService manages durable cron schedule definitions.
type ScheduleService interface {
	Create(ctx context.Context, input CreateScheduleInput) (*models.Schedule, error)
	Get(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	ListByAgent(ctx context.Context, agentID uuid.UUID) ([]models.Schedule, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateScheduleInput) (*models.Schedule, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type scheduleService struct{ repository repository.ScheduleRepository }

// NewScheduleService creates a schedule application service.
func NewScheduleService(scheduleRepository repository.ScheduleRepository) ScheduleService {
	return &scheduleService{repository: scheduleRepository}
}

func (s *scheduleService) Create(ctx context.Context, input CreateScheduleInput) (*models.Schedule, error) {
	if err := validateSchedule(input.AgentID, input.CronExpression); err != nil {
		return nil, err
	}
	schedule := &models.Schedule{
		ID:             uuid.New(),
		AgentID:        input.AgentID,
		WorkflowID:     input.WorkflowID,
		CronExpression: strings.TrimSpace(input.CronExpression),
		Enabled:        input.Enabled,
	}
	if err := s.repository.Create(ctx, schedule); err != nil {
		return nil, fmt.Errorf("create schedule service: %w", err)
	}
	return schedule, nil
}

func (s *scheduleService) Get(ctx context.Context, id uuid.UUID) (*models.Schedule, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: schedule ID is required", ErrInvalidInput)
	}
	schedule, err := s.repository.GetByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get schedule service: %w", err)
	}
	return schedule, nil
}

func (s *scheduleService) ListByAgent(ctx context.Context, agentID uuid.UUID) ([]models.Schedule, error) {
	if agentID == uuid.Nil {
		return nil, fmt.Errorf("%w: agent ID is required", ErrInvalidInput)
	}
	schedules, err := s.repository.ListByAgent(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("list schedules service: %w", err)
	}
	return schedules, nil
}

func (s *scheduleService) Update(ctx context.Context, id uuid.UUID, input UpdateScheduleInput) (*models.Schedule, error) {
	schedule, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if input.WorkflowID != nil {
		schedule.WorkflowID = input.WorkflowID
	}
	if input.CronExpression != nil {
		if err := validateCron(*input.CronExpression); err != nil {
			return nil, err
		}
		schedule.CronExpression = strings.TrimSpace(*input.CronExpression)
	}
	if input.Enabled != nil {
		schedule.Enabled = *input.Enabled
	}
	if err := s.repository.Update(ctx, schedule); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("update schedule service: %w", err)
	}
	return schedule, nil
}

func (s *scheduleService) Delete(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("%w: schedule ID is required", ErrInvalidInput)
	}
	if err := s.repository.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrNotFound
		}
		return fmt.Errorf("delete schedule service: %w", err)
	}
	return nil
}

func validateSchedule(agentID uuid.UUID, expression string) error {
	if agentID == uuid.Nil {
		return fmt.Errorf("%w: agent ID is required", ErrInvalidInput)
	}
	return validateCron(expression)
}

func validateCron(expression string) error {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.SecondOptional)
	if _, err := parser.Parse(strings.TrimSpace(expression)); err != nil {
		return fmt.Errorf("%w: invalid cron expression", ErrInvalidInput)
	}
	return nil
}
