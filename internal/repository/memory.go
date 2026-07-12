package repository

import (
	"context"
	"fmt"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MemoryRepository persists agent memory entries.
type MemoryRepository interface {
	Get(ctx context.Context, agentID uuid.UUID, key string) (*models.AgentMemory, error)
	Upsert(ctx context.Context, memory *models.AgentMemory) error
	Delete(ctx context.Context, agentID uuid.UUID, key string) error
}

type gormMemoryRepository struct{ db *gorm.DB }

// NewMemoryRepository creates a PostgreSQL-backed memory repository.
func NewMemoryRepository(db *gorm.DB) MemoryRepository {
	return &gormMemoryRepository{db: db}
}

func (r *gormMemoryRepository) Get(ctx context.Context, agentID uuid.UUID, key string) (*models.AgentMemory, error) {
	var memory models.AgentMemory
	if err := r.db.WithContext(ctx).Where("agent_id = ? AND memory_key = ?", agentID, key).First(&memory).Error; err != nil {
		return nil, wrapQueryError("get agent memory", err)
	}
	return &memory, nil
}

func (r *gormMemoryRepository) Upsert(ctx context.Context, memory *models.AgentMemory) error {
	var existing models.AgentMemory
	err := r.db.WithContext(ctx).
		Where("agent_id = ? AND memory_key = ?", memory.AgentID, memory.MemoryKey).
		First(&existing).Error
	if err == nil {
		result := r.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
			"memory_value": memory.MemoryValue,
			"updated_at":   memory.UpdatedAt,
		})
		if result.Error != nil {
			return fmt.Errorf("update agent memory: %w", result.Error)
		}
		memory.ID = existing.ID
		return nil
	}
	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("find agent memory: %w", err)
	}
	if err := r.db.WithContext(ctx).Create(memory).Error; err != nil {
		return fmt.Errorf("create agent memory: %w", err)
	}
	return nil
}

func (r *gormMemoryRepository) Delete(ctx context.Context, agentID uuid.UUID, key string) error {
	result := r.db.WithContext(ctx).Where("agent_id = ? AND memory_key = ?", agentID, key).Delete(&models.AgentMemory{})
	if result.Error != nil {
		return fmt.Errorf("delete agent memory: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
