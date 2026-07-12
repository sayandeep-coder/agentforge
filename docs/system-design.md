# System Design

## Purpose

This document describes AgentForge as a durable execution platform for autonomous AI agents. The design treats an agent run as a task with observable state, not as a synchronous chat request.

## Design goals

- Execute thousands of concurrent tasks with bounded worker resources.
- Persist task, workflow, memory, and audit state in PostgreSQL.
- Keep business logic independent of transport, storage, queue, and model vendors.
- Support retries, cancellation, graceful shutdown, and structured observability.
- Allow the initial in-memory queue to be replaced by Redis, NATS, RabbitMQ, or Kafka.

## Domain model

The core aggregate relationships are:

```mermaid
erDiagram
    AGENT ||--o{ WORKFLOW : owns
    AGENT ||--o{ TASK : executes
    AGENT ||--o{ AGENT_MEMORY : stores
    AGENT ||--o{ SCHEDULE : triggers
    WORKFLOW ||--o{ WORKFLOW_STEP : contains
    WORKFLOW ||--o{ TASK : selects
    WORKFLOW_STEP ||--o{ TASK_STEP : materializes
    TOOL ||--o{ WORKFLOW_STEP : supports
    TASK ||--o{ TASK_STEP : records
    TASK ||--o{ TASK_LOG : produces
    TASK ||--o| WORKER : claimed_by

    AGENT { uuid id PK string name string model string status }
    WORKFLOW { uuid id PK uuid agent_id FK int version bool is_active }
    WORKFLOW_STEP { uuid id PK uuid workflow_id FK string step_type int step_order }
    TASK { uuid id PK uuid agent_id FK uuid workflow_id FK string status int priority json input }
    TASK_STEP { uuid id PK uuid task_id FK uuid workflow_step_id FK string status int latency_ms }
    TOOL { uuid id PK string name UK string type bool is_enabled json config }
    AGENT_MEMORY { uuid id PK uuid agent_id FK string memory_key json memory_value }
    TASK_LOG { uuid id PK uuid task_id FK string level string message json metadata }
    WORKER { uuid id PK string worker_name UK string status uuid current_task_id FK }
    SCHEDULE { uuid id PK uuid agent_id FK uuid workflow_id FK string cron_expression bool enabled }
```

PostgreSQL owns durable state. `tasks` and `task_steps` are the execution record; `task_logs` is the append-oriented operational history. Foreign keys protect lifecycle integrity when agents, workflows, or tasks are removed.

## Task execution lifecycle

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant API as HTTP API
    participant S as Task Service
    participant DB as PostgreSQL
    participant Q as Queue
    participant W as Worker Pool
    participant R as Tool/LLM Runtime

    C->>API: POST /v1/tasks
    API->>S: Validate and submit command
    S->>DB: Transaction: create task (pending)
    S->>Q: Publish task ID and priority
    S-->>API: Task reference
    API-->>C: 202 Accepted
    W->>Q: Claim next task
    W->>DB: Mark running; create task-step record
    W->>R: Execute step with context and timeout
    R-->>W: Result or typed error
    W->>DB: Persist step result and task status
    W->>DB: Append structured task log
    C->>API: GET /v1/tasks/{taskID}
    API->>DB: Read task projection
    DB-->>API: Status, output, error
    API-->>C: 200 OK
```

The API acknowledges accepted work quickly. Workers own execution and state transitions. Every transition must be idempotent or protected by a claim/lease rule so a retry cannot produce an invalid terminal state.

## Boundaries and interfaces

| Boundary | Responsibility | Replaceable dependency |
| --- | --- | --- |
| Domain | Agent, workflow, task, step, memory rules | None |
| Application services | Commands, queries, transactions, orchestration | Repositories and ports |
| Queue port | Publish and consume task messages | In-memory, Redis, NATS, RabbitMQ, Kafka |
| Worker runtime | Concurrency, retry, cancellation, shutdown | Queue and execution ports |
| Tool registry | Resolve and invoke named tools | Tool implementations |
| LLM port | Generate model output | OpenAI, Gemini, Anthropic, Ollama, mock |
| Repository port | Durable reads and writes | GORM/PostgreSQL implementation |
| Delivery | HTTP routing, validation, serialization | Gin |

## Reliability decisions

- A task starts as `pending`, becomes `running` after a worker claim, and ends in `completed`, `failed`, or `cancelled`.
- Retry policy belongs to the worker/application boundary, not to a specific tool or HTTP handler.
- Context cancellation propagates from shutdown or task cancellation into each step and provider call.
- Queue delivery is treated as at-least-once; task execution therefore requires idempotency.
- PostgreSQL transactions cover coupled state changes such as task creation and task-step materialization.
- Redis is an acceleration and coordination layer, not the authoritative source of task history.

## Security and governance

Credentials belong in environment or secret-manager configuration, never in agent or tool payloads. API authentication, authorization, rate limiting, and tenant isolation should be added at the delivery boundary before exposing the service publicly. Tool configuration must be validated and access-controlled because tools are an execution boundary.
