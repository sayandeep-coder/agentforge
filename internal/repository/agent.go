package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AgentRepository persists agents.
type AgentRepository interface {
	Create(ctx context.Context, agent *models.Agent) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	List(ctx context.Context, status models.AgentStatus, offset, limit int) ([]models.Agent, int64, error)
	Update(ctx context.Context, agent *models.Agent) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type gormAgentRepository struct{ db *gorm.DB }

// NewAgentRepository creates a PostgreSQL-backed agent repository.
func NewAgentRepository(db *gorm.DB) AgentRepository {
	return &gormAgentRepository{db: db}
}

func (r *gormAgentRepository) Create(ctx context.Context, agent *models.Agent) error {
	if err := r.db.WithContext(ctx).Create(agent).Error; err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	return nil
}

func (r *gormAgentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Agent, error) {
	var agent models.Agent
	if err := r.db.WithContext(ctx).First(&agent, "id = ?", id).Error; err != nil {
		return nil, wrapQueryError("get agent", err)
	}
	return &agent, nil
}

func (r *gormAgentRepository) List(ctx context.Context, status models.AgentStatus, offset, limit int) ([]models.Agent, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Agent{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count agents: %w", err)
	}

	var agents []models.Agent
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&agents).Error; err != nil {
		return nil, 0, fmt.Errorf("list agents: %w", err)
	}
	return agents, total, nil
}

func (r *gormAgentRepository) Update(ctx context.Context, agent *models.Agent) error {
	result := r.db.WithContext(ctx).Model(&models.Agent{}).Where("id = ?", agent.ID).Updates(agent)
	if result.Error != nil {
		return fmt.Errorf("update agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormAgentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).Delete(&models.Agent{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func wrapQueryError(operation string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}
