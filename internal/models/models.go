// Package models contains persistence models that map to the AgentForge schema.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AgentStatus identifies the lifecycle state of an agent.
type AgentStatus string

const (
	// AgentStatusActive indicates that an agent may execute tasks.
	AgentStatusActive AgentStatus = "active"
	// AgentStatusInactive indicates that an agent is retained but cannot execute new tasks.
	AgentStatusInactive AgentStatus = "inactive"
)

// TaskStatus identifies the lifecycle state of a task.
type TaskStatus string

const (
	// TaskStatusPending identifies a task waiting for a worker.
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusRunning identifies a task currently being executed.
	TaskStatusRunning TaskStatus = "running"
	// TaskStatusCompleted identifies a successful task.
	TaskStatusCompleted TaskStatus = "completed"
	// TaskStatusFailed identifies a task that terminated with an error.
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusCancelled identifies a task stopped by cancellation.
	TaskStatusCancelled TaskStatus = "cancelled"
)

// StepStatus identifies the lifecycle state of a task step.
type StepStatus string

const (
	// StepStatusPending identifies a step that has not started.
	StepStatusPending StepStatus = "pending"
	// StepStatusRunning identifies a step currently executing.
	StepStatusRunning StepStatus = "running"
	// StepStatusCompleted identifies a successful step.
	StepStatusCompleted StepStatus = "completed"
	// StepStatusFailed identifies a step that terminated with an error.
	StepStatusFailed StepStatus = "failed"
	// StepStatusCancelled identifies a step stopped by cancellation.
	StepStatusCancelled StepStatus = "cancelled"
)

// Agent maps to the agents table.
type Agent struct {
	ID           uuid.UUID   `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name         string      `gorm:"size:100;not null"`
	Description  string      `gorm:"type:text"`
	SystemPrompt string      `gorm:"type:text;not null"`
	Model        string      `gorm:"size:100;not null"`
	Temperature  float64     `gorm:"type:decimal(3,2);default:0.7"`
	MaxTokens    int         `gorm:"default:4096"`
	Status       AgentStatus `gorm:"size:20;default:active;index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// TableName returns the database table name for Agent.
func (Agent) TableName() string { return "agents" }

// Workflow maps to the workflows table.
type Workflow struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AgentID     uuid.UUID `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"size:100;not null"`
	Description string    `gorm:"type:text"`
	Version     int       `gorm:"default:1"`
	IsActive    bool      `gorm:"default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Steps       []WorkflowStep `gorm:"foreignKey:WorkflowID"`
}

// TableName returns the database table name for Workflow.
func (Workflow) TableName() string { return "workflows" }

// Tool maps to the tools table.
type Tool struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name        string         `gorm:"size:100;unique;not null"`
	Description string         `gorm:"type:text"`
	Type        string         `gorm:"size:50;not null"`
	Endpoint    string         `gorm:"type:text"`
	IsEnabled   bool           `gorm:"default:true"`
	Config      datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	CreatedAt   time.Time
}

// TableName returns the database table name for Tool.
func (Tool) TableName() string { return "tools" }

// WorkflowStep maps to the workflow_steps table.
type WorkflowStep struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	WorkflowID uuid.UUID      `gorm:"type:uuid;not null;index"`
	StepName   string         `gorm:"size:100;not null"`
	StepType   string         `gorm:"size:50;not null"`
	StepOrder  int            `gorm:"not null"`
	ToolID     *uuid.UUID     `gorm:"type:uuid"`
	NextStepID *uuid.UUID     `gorm:"type:uuid"`
	Config     datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	CreatedAt  time.Time
}

// TableName returns the database table name for WorkflowStep.
func (WorkflowStep) TableName() string { return "workflow_steps" }

// Task maps to the tasks table.
type Task struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AgentID     uuid.UUID      `gorm:"type:uuid;not null;index"`
	WorkflowID  *uuid.UUID     `gorm:"type:uuid;index"`
	Status      TaskStatus     `gorm:"size:20;not null;default:pending;index"`
	Priority    int            `gorm:"default:0"`
	Input       datatypes.JSON `gorm:"type:jsonb;not null"`
	Output      datatypes.JSON `gorm:"type:jsonb"`
	Error       string         `gorm:"type:text"`
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	Steps       []TaskStep `gorm:"foreignKey:TaskID"`
	Logs        []TaskLog  `gorm:"foreignKey:TaskID"`
}

// TableName returns the database table name for Task.
func (Task) TableName() string { return "tasks" }

// TaskStep maps to the task_steps table.
type TaskStep struct {
	ID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TaskID         uuid.UUID      `gorm:"type:uuid;not null;index"`
	WorkflowStepID *uuid.UUID     `gorm:"type:uuid"`
	Status         StepStatus     `gorm:"size:20;default:pending"`
	Input          datatypes.JSON `gorm:"type:jsonb"`
	Output         datatypes.JSON `gorm:"type:jsonb"`
	LatencyMS      int            `gorm:"column:latency_ms"`
	StartedAt      *time.Time
	CompletedAt    *time.Time
}

// TableName returns the database table name for TaskStep.
func (TaskStep) TableName() string { return "task_steps" }

// AgentMemory maps to the agent_memory table.
type AgentMemory struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AgentID     uuid.UUID      `gorm:"type:uuid;not null;index"`
	MemoryKey   string         `gorm:"size:255;not null"`
	MemoryValue datatypes.JSON `gorm:"type:jsonb;not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TableName returns the database table name for AgentMemory.
func (AgentMemory) TableName() string { return "agent_memory" }

// TaskLog maps to the task_logs table.
type TaskLog struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	TaskID    uuid.UUID      `gorm:"type:uuid;not null;index"`
	Level     string         `gorm:"size:20;default:INFO"`
	Message   string         `gorm:"type:text;not null"`
	Metadata  datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	CreatedAt time.Time
}

// TableName returns the database table name for TaskLog.
func (TaskLog) TableName() string { return "task_logs" }

// Worker maps to the workers table.
type Worker struct {
	ID            uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	WorkerName    string     `gorm:"size:100;unique;not null"`
	Status        string     `gorm:"size:20;default:idle;index"`
	CurrentTaskID *uuid.UUID `gorm:"type:uuid"`
	LastHeartbeat *time.Time
	CreatedAt     time.Time
}

// TableName returns the database table name for Worker.
func (Worker) TableName() string { return "workers" }

// Schedule maps to the schedules table.
type Schedule struct {
	ID             uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	AgentID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	WorkflowID     *uuid.UUID `gorm:"type:uuid"`
	CronExpression string     `gorm:"size:100;not null"`
	Enabled        bool       `gorm:"default:true;index"`
	LastRun        *time.Time
	NextRun        *time.Time
	CreatedAt      time.Time
}

// TableName returns the database table name for Schedule.
func (Schedule) TableName() string { return "schedules" }
