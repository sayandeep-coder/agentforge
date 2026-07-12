# System Architecture

## Logical architecture

The system is organized into four boundaries: delivery, application, execution, and infrastructure. Arrows represent dependency direction; the domain does not depend on adapters.

```mermaid
flowchart LR
    classDef client fill:#E8F0FE,stroke:#1A73E8,color:#174EA6
    classDef delivery fill:#E6F4EA,stroke:#188038,color:#137333
    classDef app fill:#FEF7E0,stroke:#F9AB00,color:#8D5A00
    classDef exec fill:#F3E8FD,stroke:#9334E6,color:#681DA8
    classDef data fill:#FCE8E6,stroke:#D93025,color:#A50E0E
    classDef external fill:#F1F3F4,stroke:#5F6368,color:#202124

    subgraph Clients[Clients and operators]
        SDK[SDK / CLI]
        UI[Operations Console]
    end

    subgraph Delivery[Delivery boundary]
        HTTP[Gin HTTP API]
        MW[Request ID • CORS • Recovery • Logging]
    end

    subgraph Application[Application boundary]
        AS[Application Services]
        VAL[Validation and typed errors]
        REPO[Repository interfaces]
    end

    subgraph Execution[Execution boundary]
        QUEUE[Queue port]
        POOL[Worker Pool]
        ORCH[Workflow Orchestrator]
        TOOLS[Tool Registry]
        LLM[LLM Provider port]
        MEM[Memory Service]
        SCHED[Scheduler]
    end

    subgraph Infrastructure[Infrastructure adapters]
        PG[(PostgreSQL)]
        REDIS[(Redis)]
        LOG[Zap Structured Logs]
    end

    EXT[LLM vendors and external tools]

    SDK -->|REST / JSON| HTTP
    UI -->|REST / JSON| HTTP
    HTTP --> MW --> VAL --> AS
    AS --> REPO -->|GORM| PG
    AS -->|publish task| QUEUE
    QUEUE --> POOL --> ORCH
    ORCH --> TOOLS --> EXT
    ORCH --> LLM --> EXT
    ORCH --> MEM --> REPO
    SCHED --> AS
    QUEUE -.->|optional adapter| REDIS
    MW -.-> LOG
    POOL -.-> LOG

    class SDK,UI client
    class HTTP,MW delivery
    class AS,VAL,REPO app
    class QUEUE,POOL,ORCH,TOOLS,LLM,MEM,SCHED exec
    class PG,REDIS,LOG data
    class EXT external
```

## Runtime and deployment topology

The first deployment can run as one Go service with an in-memory queue. The interfaces preserve the path to a horizontally scaled API/worker deployment with Redis-backed coordination.

```mermaid
flowchart TB
    classDef edge fill:#E8F0FE,stroke:#1A73E8,color:#174EA6
    classDef compute fill:#E6F4EA,stroke:#188038,color:#137333
    classDef state fill:#FCE8E6,stroke:#D93025,color:#A50E0E
    classDef provider fill:#F1F3F4,stroke:#5F6368,color:#202124

    Internet[Clients / internal services]:::edge --> LB[HTTPS load balancer]:::edge

    subgraph AgentForge[AgentForge service boundary]
        API1[API instance]:::compute
        API2[API instance]:::compute
        W1[Worker process / pool]:::compute
        W2[Worker process / pool]:::compute
        CRON[Scheduler instance]:::compute
    end

    LB --> API1
    LB --> API2
    API1 -->|commands and queries| DB[(Neon PostgreSQL)]:::state
    API2 -->|commands and queries| DB
    API1 -->|enqueue| Q[(Redis queue)]:::state
    API2 -->|enqueue| Q
    Q --> W1
    Q --> W2
    CRON -->|scheduled task submission| DB
    CRON -->|enqueue| Q
    W1 -->|claims and persists| DB
    W2 -->|claims and persists| DB
    W1 -->|provider calls| PROVIDERS[LLM providers / tools]:::provider
    W2 -->|provider calls| PROVIDERS

    DEV[Local development mode]:::provider -.->|replace Redis adapter| MEM[(In-memory queue)]:::state
```

## Request and data flow

1. The load balancer terminates TLS and forwards requests to an API instance.
2. Middleware assigns a request ID, recovers panics, applies CORS policy, and emits structured access logs.
3. Application services validate commands and execute repository operations inside explicit transactions.
4. Task submission writes the durable task record before publishing work. A recovery/reconciliation process can identify persisted pending tasks that were not published.
5. Workers claim work, execute steps with bounded concurrency and context deadlines, then persist outputs and logs.
6. Reads use PostgreSQL as the source of truth. Redis may cache hot data or coordinate distributed queue consumption.

## Scalability and failure model

| Concern | Strategy |
| --- | --- |
| API scale | Stateless API instances behind a load balancer |
| Worker scale | Increase worker replicas and configure per-process pool size |
| Queue durability | Start in-memory; use Redis or a durable broker for multi-instance deployments |
| Database pressure | Connection pooling, indexed task queries, bounded worker concurrency |
| Provider slowness | Context deadlines, retry budgets, circuit-breaking at the provider adapter |
| Process failure | At-least-once delivery with task claim/lease and idempotent transitions |
| Shutdown | Stop intake, drain or cancel workers, close queue and database resources |
| Observability | Correlated structured logs, task logs, latency, status, and worker identity |

## Package mapping

```text
cmd/server          composition root and process lifecycle
internal/api         HTTP handlers, routes, request/response DTOs
internal/agent       agent domain and service
internal/workflow    workflow and step domain
internal/worker      worker pool, retries, claims, shutdown
internal/queue       queue interface and adapters
internal/scheduler   cron scheduling and task submission
internal/memory      persistent memory service
internal/tools       tool interface and registry
internal/llm         LLM provider interface and adapters
internal/repository  persistence interfaces and GORM implementations
internal/database    PostgreSQL connection and transactions
internal/models      persistence models
internal/middleware  cross-cutting HTTP concerns
internal/config      environment-backed configuration
internal/logger      Zap construction and fields
```
