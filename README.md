# AgentForge

AgentForge is an AI agent infrastructure platform written in Go. It provides the runtime primitives needed to define, execute, orchestrate, observe, and schedule autonomous agents. It is designed as infrastructure—not as a conversational chatbot or a user-facing chat application.

The platform is intended to combine the durable execution model of Temporal, the workload isolation model of Kubernetes, and the event-driven execution model of AWS Lambda for AI-agent workloads.

## Why AgentForge

AI agents need more than an LLM call. Production systems must manage agent configuration, workflow steps, tools, memory, task state, retries, workers, schedules, and operational telemetry as one coherent execution system.

AgentForge separates those concerns behind stable interfaces so that infrastructure can evolve without rewriting business logic. PostgreSQL is the source of truth for durable state, Redis is reserved for distributed caching and queue implementations, and the execution plane is decoupled from HTTP handlers through services, repositories, queues, and workers.

## How it works

1. A client creates or selects an agent and submits a task or workflow execution request.
2. The API validates the request and passes it to the application service layer.
3. The service persists the task and publishes a queue message.
4. A worker claims the task, executes workflow steps, invokes tools or an LLM provider, and records state and logs.
5. The client reads task status and results through the API. Schedules can submit tasks automatically.

The project follows Clean Architecture and Domain-Driven Design. Domain and application logic do not depend on Gin, GORM, PostgreSQL, Redis, or a specific LLM vendor.

## Project status

The current implementation includes configuration, logging, PostgreSQL/GORM access, schema-aligned models, repositories, services, queue and worker runtime, tool and LLM ports, cron scheduling, middleware, and versioned REST endpoints. The supplied PostgreSQL schema is preserved in `migrations/000001_initial_schema.sql`.

## Configuration

Copy the local environment template values into `.env` and add the database and Redis connection URLs:

```env
APP_ENV=development
SERVER_PORT=8080
DATABASE_URL=postgres://user:password@host:5432/agentforge?sslmode=require
REDIS_URL=redis://user:password@host:6379/0
```

Never commit real credentials. The database schema is included in `migrations/000001_initial_schema.sql` and must exist before the application starts.

## Documentation

- [System Design](docs/system-design.md) — domain model, execution lifecycle, boundaries, and design decisions.
- [System Architecture](docs/system-architecture.md) — logical components, runtime topology, deployment concerns, and operational flows.
- [API Documentation](docs/api-docs.md) — REST conventions, endpoint contract, request examples, and task lifecycle.

## Run

After setting `DATABASE_URL` and `REDIS_URL` in `.env`:

```bash
go run ./cmd/server
```

The server exposes liveness at `/healthz`, readiness at `/readyz`, and versioned application routes under `/v1`.

## Engineering principles

- Context-aware operations and graceful shutdown.
- Structured logging with request, worker, agent, and task correlation.
- Repository and provider interfaces for replaceable infrastructure.
- Explicit validation, typed errors, transactions, and consistent API responses.
- Small packages with dependency injection and testable business logic.

## License

See [LICENSE](LICENSE).
