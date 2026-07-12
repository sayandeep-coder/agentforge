// Package agent provides the task execution runtime.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/agentforge/agentforge/internal/llm"
	"github.com/agentforge/agentforge/internal/models"
	"github.com/agentforge/agentforge/internal/queue"
	"github.com/agentforge/agentforge/internal/repository"
	"github.com/agentforge/agentforge/internal/tools"
	"github.com/agentforge/agentforge/internal/workflow"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// Runtime executes a queued task and persists its terminal state.
type Runtime struct {
	agents    repository.AgentRepository
	workflows repository.WorkflowRepository
	tasks     repository.TaskRepository
	engine    *workflow.Engine
	logger    *zap.Logger
}

// Config contains dependencies for the agent runtime.
type Config struct {
	Agents       repository.AgentRepository
	Workflows    repository.WorkflowRepository
	Tasks        repository.TaskRepository
	ToolRegistry tools.Registry
	LLMProvider  llm.LLMProvider
	Logger       *zap.Logger
}

// NewRuntime creates an executor for queued task messages.
func NewRuntime(config Config) (*Runtime, error) {
	if config.Agents == nil || config.Workflows == nil || config.Tasks == nil {
		return nil, errors.New("agent runtime repositories are required")
	}
	if config.ToolRegistry == nil {
		return nil, errors.New("agent runtime tool registry is required")
	}
	engine, err := workflow.NewEngine(config.ToolRegistry, config.LLMProvider)
	if err != nil {
		return nil, fmt.Errorf("create workflow engine: %w", err)
	}
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	return &Runtime{agents: config.Agents, workflows: config.Workflows, tasks: config.Tasks, engine: engine, logger: config.Logger}, nil
}

// Execute claims and executes one queued task.
func (r *Runtime) Execute(ctx context.Context, message queue.Message) error {
	task, err := r.tasks.GetByID(ctx, message.TaskID)
	if err != nil {
		return fmt.Errorf("load task %s: %w", message.TaskID, err)
	}
	if task.Status == models.TaskStatusCompleted || task.Status == models.TaskStatusCancelled {
		return nil
	}
	if task.Status == models.TaskStatusPending {
		if err := r.tasks.UpdateStatus(ctx, task.ID, models.TaskStatusPending, models.TaskStatusRunning); err != nil {
			return fmt.Errorf("claim task %s: %w", task.ID, err)
		}
		task.Status = models.TaskStatusRunning
	}

	startedAt := time.Now().UTC()
	task.StartedAt = &startedAt
	output, executeErr := r.executeTask(ctx, task)
	completedAt := time.Now().UTC()
	task.CompletedAt = &completedAt
	if executeErr != nil {
		task.Status = models.TaskStatusFailed
		task.Error = executeErr.Error()
		r.persistTerminalState(ctx, task)
		return nil
	}
	task.Status = models.TaskStatusCompleted
	task.Output = datatypes.JSON(output)
	if err := r.tasks.Update(ctx, task); err != nil {
		return fmt.Errorf("persist completed task %s: %w", task.ID, err)
	}
	return nil
}

func (r *Runtime) executeTask(ctx context.Context, task *models.Task) (json.RawMessage, error) {
	agentDefinition, err := r.agents.GetByID(ctx, task.AgentID)
	if err != nil {
		return nil, fmt.Errorf("load agent: %w", err)
	}
	if task.WorkflowID == nil {
		return r.engine.Execute(ctx, agentDefinition, &models.Workflow{Steps: []models.WorkflowStep{{StepName: "direct", StepType: "llm", StepOrder: 1}}}, json.RawMessage(task.Input))
	}
	workflowDefinition, err := r.workflows.GetByID(ctx, *task.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("load workflow: %w", err)
	}
	return r.engine.Execute(ctx, agentDefinition, workflowDefinition, json.RawMessage(task.Input))
}

func (r *Runtime) persistTerminalState(ctx context.Context, task *models.Task) {
	if err := r.tasks.Update(ctx, task); err != nil {
		r.logger.Error("failed to persist failed task", zap.String("task_id", task.ID.String()), zap.Error(err))
	}
}
