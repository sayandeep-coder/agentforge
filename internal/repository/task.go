package repository

import (
	"context"
	"fmt"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TaskRepository persists task execution state.
type TaskRepository interface {
	Create(ctx context.Context, task *models.Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error)
	Update(ctx context.Context, task *models.Task) error
	UpdateStatus(ctx context.Context, id uuid.UUID, from, to models.TaskStatus) error
}

type gormTaskRepository struct{ db *gorm.DB }

// NewTaskRepository creates a PostgreSQL-backed task repository.
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &gormTaskRepository{db: db}
}

func (r *gormTaskRepository) Create(ctx context.Context, task *models.Task) error {
	if err := r.db.WithContext(ctx).Create(task).Error; err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

func (r *gormTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	var task models.Task
	query := r.db.WithContext(ctx).Preload("Steps").Preload("Logs", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at ASC")
	})
	if err := query.First(&task, "id = ?", id).Error; err != nil {
		return nil, wrapQueryError("get task", err)
	}
	return &task, nil
}

func (r *gormTaskRepository) Update(ctx context.Context, task *models.Task) error {
	result := r.db.WithContext(ctx).Model(&models.Task{}).Where("id = ?", task.ID).Select("*").Updates(task)
	if result.Error != nil {
		return fmt.Errorf("update task: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, from, to models.TaskStatus) error {
	result := r.db.WithContext(ctx).Model(&models.Task{}).
		Where("id = ? AND status = ?", id, from).
		Update("status", to)
	if result.Error != nil {
		return fmt.Errorf("update task status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}
