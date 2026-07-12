package repository

import (
	"context"
	"fmt"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkflowRepository persists workflows and their ordered steps.
type WorkflowRepository interface {
	Create(ctx context.Context, workflow *models.Workflow) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error)
	ListByAgent(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]models.Workflow, int64, error)
	Update(ctx context.Context, workflow *models.Workflow) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type gormWorkflowRepository struct{ db *gorm.DB }

// NewWorkflowRepository creates a PostgreSQL-backed workflow repository.
func NewWorkflowRepository(db *gorm.DB) WorkflowRepository {
	return &gormWorkflowRepository{db: db}
}

func (r *gormWorkflowRepository) Create(ctx context.Context, workflow *models.Workflow) error {
	if err := r.db.WithContext(ctx).Create(workflow).Error; err != nil {
		return fmt.Errorf("create workflow: %w", err)
	}
	return nil
}

func (r *gormWorkflowRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Workflow, error) {
	var workflow models.Workflow
	query := r.db.WithContext(ctx).Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_order ASC")
	})
	if err := query.First(&workflow, "id = ?", id).Error; err != nil {
		return nil, wrapQueryError("get workflow", err)
	}
	return &workflow, nil
}

func (r *gormWorkflowRepository) ListByAgent(ctx context.Context, agentID uuid.UUID, offset, limit int) ([]models.Workflow, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Workflow{}).Where("agent_id = ?", agentID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count workflows: %w", err)
	}

	var workflows []models.Workflow
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&workflows).Error; err != nil {
		return nil, 0, fmt.Errorf("list workflows: %w", err)
	}
	return workflows, total, nil
}

func (r *gormWorkflowRepository) Update(ctx context.Context, workflow *models.Workflow) error {
	result := r.db.WithContext(ctx).Model(&models.Workflow{}).Where("id = ?", workflow.ID).Updates(workflow)
	if result.Error != nil {
		return fmt.Errorf("update workflow: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormWorkflowRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Workflow{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete workflow: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
