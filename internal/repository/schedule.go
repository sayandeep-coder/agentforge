package repository

import (
	"context"
	"fmt"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ScheduleRepository persists cron schedule definitions.
type ScheduleRepository interface {
	Create(ctx context.Context, schedule *models.Schedule) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	ListByAgent(ctx context.Context, agentID uuid.UUID) ([]models.Schedule, error)
	Update(ctx context.Context, schedule *models.Schedule) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type gormScheduleRepository struct{ db *gorm.DB }

// NewScheduleRepository creates a PostgreSQL-backed schedule repository.
func NewScheduleRepository(db *gorm.DB) ScheduleRepository {
	return &gormScheduleRepository{db: db}
}

func (r *gormScheduleRepository) Create(ctx context.Context, schedule *models.Schedule) error {
	if err := r.db.WithContext(ctx).Create(schedule).Error; err != nil {
		return fmt.Errorf("create schedule: %w", err)
	}
	return nil
}

func (r *gormScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error) {
	var schedule models.Schedule
	if err := r.db.WithContext(ctx).First(&schedule, "id = ?", id).Error; err != nil {
		return nil, wrapQueryError("get schedule", err)
	}
	return &schedule, nil
}

func (r *gormScheduleRepository) ListByAgent(ctx context.Context, agentID uuid.UUID) ([]models.Schedule, error) {
	var schedules []models.Schedule
	if err := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Order("created_at DESC").Find(&schedules).Error; err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	return schedules, nil
}

func (r *gormScheduleRepository) Update(ctx context.Context, schedule *models.Schedule) error {
	result := r.db.WithContext(ctx).Model(&models.Schedule{}).Where("id = ?", schedule.ID).Select("*").Updates(schedule)
	if result.Error != nil {
		return fmt.Errorf("update schedule: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Schedule{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete schedule: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
