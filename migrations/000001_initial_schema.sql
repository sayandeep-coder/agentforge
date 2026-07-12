CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(100) NOT NULL,
    description TEXT, system_prompt TEXT NOT NULL, model VARCHAR(100) NOT NULL,
    temperature DECIMAL(3,2) DEFAULT 0.7, max_tokens INTEGER DEFAULT 4096,
    status VARCHAR(20) DEFAULT 'active', created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL, description TEXT, version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT TRUE, created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT, type VARCHAR(50) NOT NULL, endpoint TEXT,
    is_enabled BOOLEAN DEFAULT TRUE, config JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE workflow_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    step_name VARCHAR(100) NOT NULL, step_type VARCHAR(50) NOT NULL,
    step_order INTEGER NOT NULL, tool_id UUID REFERENCES tools(id) ON DELETE SET NULL,
    next_step_id UUID, config JSONB DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE workflow_steps ADD CONSTRAINT fk_next_step FOREIGN KEY (next_step_id)
REFERENCES workflow_steps(id) ON DELETE SET NULL;

CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', priority INTEGER DEFAULT 0,
    input JSONB NOT NULL, output JSONB, error TEXT, created_at TIMESTAMPTZ DEFAULT NOW(),
    started_at TIMESTAMPTZ, completed_at TIMESTAMPTZ
);

CREATE TABLE task_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    workflow_step_id UUID REFERENCES workflow_steps(id) ON DELETE SET NULL,
    status VARCHAR(20) DEFAULT 'pending', input JSONB, output JSONB,
    latency_ms INTEGER, started_at TIMESTAMPTZ, completed_at TIMESTAMPTZ
);

CREATE TABLE agent_memory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    memory_key VARCHAR(255) NOT NULL, memory_value JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(), updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE task_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    level VARCHAR(20) DEFAULT 'INFO', message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}'::jsonb, created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE workers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), worker_name VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'idle', current_task_id UUID REFERENCES tasks(id) ON DELETE SET NULL,
    last_heartbeat TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    cron_expression VARCHAR(100) NOT NULL, enabled BOOLEAN DEFAULT TRUE,
    last_run TIMESTAMPTZ, next_run TIMESTAMPTZ, created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_agents_status ON agents(status);
CREATE INDEX idx_workflows_agent_id ON workflows(agent_id);
CREATE INDEX idx_workflow_steps_workflow_id ON workflow_steps(workflow_id);
CREATE INDEX idx_tasks_agent_id ON tasks(agent_id);
CREATE INDEX idx_tasks_workflow_id ON tasks(workflow_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);
CREATE INDEX idx_task_steps_task_id ON task_steps(task_id);
CREATE INDEX idx_agent_memory_agent_id ON agent_memory(agent_id);
CREATE INDEX idx_task_logs_task_id ON task_logs(task_id);
CREATE INDEX idx_workers_status ON workers(status);
CREATE INDEX idx_schedules_agent_id ON schedules(agent_id);
CREATE INDEX idx_schedules_enabled ON schedules(enabled);
