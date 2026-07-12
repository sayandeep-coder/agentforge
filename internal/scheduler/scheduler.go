// Package scheduler provides cron-based task triggers.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Trigger submits work associated with a schedule ID.
type Trigger func(context.Context, uuid.UUID) error

// Scheduler manages cron expressions and their trigger callbacks.
type Scheduler struct {
	cron    *cron.Cron
	trigger Trigger
	logger  *zap.Logger

	mu      sync.RWMutex
	started bool
	ctx     context.Context
	jobs    map[uuid.UUID]cron.EntryID
}

// New creates a scheduler using five-field cron expressions with optional seconds.
func New(trigger Trigger, logger *zap.Logger) (*Scheduler, error) {
	if trigger == nil {
		return nil, errors.New("scheduler trigger is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Scheduler{
		cron: cron.New(cron.WithParser(cron.NewParser(
			cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
		))),
		trigger: trigger,
		logger:  logger,
		jobs:    make(map[uuid.UUID]cron.EntryID),
	}, nil
}

// Start begins evaluating registered schedules.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return errors.New("scheduler already started")
	}
	s.ctx = ctx
	s.started = true
	s.cron.Start()
	return nil
}

// Add registers or replaces a schedule. It may be called before or after Start.
func (s *Scheduler) Add(scheduleID uuid.UUID, expression string) error {
	if scheduleID == uuid.Nil {
		return errors.New("schedule ID is required")
	}
	if expression == "" {
		return errors.New("cron expression is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, exists := s.jobs[scheduleID]; exists {
		s.cron.Remove(existing)
	}
	entryID, err := s.cron.AddFunc(expression, func() {
		s.mu.RLock()
		ctx := s.ctx
		s.mu.RUnlock()
		if ctx == nil {
			ctx = context.Background()
		}
		if err := s.trigger(ctx, scheduleID); err != nil {
			s.logger.Error("scheduled trigger failed", zap.String("schedule_id", scheduleID.String()), zap.Error(err))
		}
	})
	if err != nil {
		return fmt.Errorf("add schedule: %w", err)
	}
	s.jobs[scheduleID] = entryID
	return nil
}

// Remove unregisters a schedule.
func (s *Scheduler) Remove(scheduleID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entryID, exists := s.jobs[scheduleID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, scheduleID)
	}
}

// Stop waits for running cron jobs to finish and releases scheduler resources.
func (s *Scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = false
	s.mu.Unlock()

	done := s.cron.Stop()
	select {
	case <-done.Done():
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop scheduler: %w", ctx.Err())
	}
}
