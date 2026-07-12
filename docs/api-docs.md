# API Documentation

## API conventions

Base URL: `http://localhost:8080`

All endpoints use JSON and are versioned under `/v1`. Responses use one envelope:

```json
{
  "success": true,
  "message": "Task accepted",
  "data": {},
  "error": null
}
```

Errors use the same envelope with `success: false`, a human-readable `message`, and a structured `error`:

```json
{
  "success": false,
  "message": "Validation failed",
  "data": null,
  "error": {
    "code": "VALIDATION_ERROR",
    "details": [{"field": "agent_id", "reason": "must be a valid UUID"}]
  }
}
```

## Health

### `GET /healthz`

Returns process liveness. It should not require database access.

### `GET /readyz`

Returns readiness after configuration, database, and required runtime dependencies are initialized.

## Agents

### `POST /v1/agents`

Creates an agent.

```json
{
  "name": "research-agent",
  "description": "Collects and summarizes research",
  "system_prompt": "You are a careful research agent.",
  "model": "mock",
  "temperature": 0.7,
  "max_tokens": 4096
}
```

Returns `201 Created`.

### `GET /v1/agents`

Lists agents. Supported query parameters: `status`, `page`, `page_size`.

### `GET /v1/agents/{agentID}`

Returns one agent or `404 Not Found`.

### `PATCH /v1/agents/{agentID}`

Updates supplied mutable fields. Returns `200 OK`.

### `DELETE /v1/agents/{agentID}`

Deletes an agent and its dependent records according to the database foreign-key policy. Returns `204 No Content`.

## Workflows

### `POST /v1/agents/{agentID}/workflows`

Creates a workflow and ordered steps. A step may invoke an LLM, a registered tool, or a control operation.

```json
{
  "name": "research-and-summarize",
  "description": "Research followed by summarization",
  "steps": [
    {"step_name": "research", "step_type": "tool", "step_order": 1, "config": {"tool": "web_search"}},
    {"step_name": "summarize", "step_type": "llm", "step_order": 2, "config": {}}
  ]
}
```

Returns `201 Created`.

### `GET /v1/agents/{agentID}/workflows`

Lists workflows for an agent.

### `GET /v1/workflows/{workflowID}`

Returns a workflow and its ordered steps.

### `PATCH /v1/workflows/{workflowID}`

Updates workflow metadata or active status.

## Tasks

### `POST /v1/tasks`

Submits asynchronous work. `workflow_id` is optional when the agent has a direct execution path.

```json
{
  "agent_id": "00000000-0000-0000-0000-000000000000",
  "workflow_id": "00000000-0000-0000-0000-000000000000",
  "priority": 10,
  "input": {"query": "Explain durable execution"}
}
```

Returns `202 Accepted` with the created task in `data`.

```bash
curl -X POST http://localhost:8080/v1/tasks \
  -H 'Content-Type: application/json' \
  -d '{"agent_id":"AGENT_ID","input":{"query":"hello"}}'
```

### `GET /v1/tasks/{taskID}`

Returns task status, output, error, timestamps, and step summaries.

### `POST /v1/tasks/{taskID}/cancel`

Requests cancellation. Returns `202 Accepted`; cancellation is cooperative and is reflected once the worker observes the context cancellation.

### `GET /v1/tasks/{taskID}/logs`

Returns structured task logs. Supported query parameters: `level`, `since`, `page`, `page_size`.

## Memory

### `PUT /v1/agents/{agentID}/memory/{key}`

Creates or updates JSON memory for an agent. Returns `200 OK`.

### `GET /v1/agents/{agentID}/memory/{key}`

Retrieves one memory item.

### `DELETE /v1/agents/{agentID}/memory/{key}`

Deletes one memory item. Returns `204 No Content`.

## Scheduling

### `POST /v1/agents/{agentID}/schedules`

Creates a cron-triggered task source.

```json
{
  "workflow_id": "WORKFLOW_ID",
  "cron_expression": "0 * * * *",
  "enabled": true
}
```

### `GET /v1/agents/{agentID}/schedules`

Lists schedules and their last and next run times.

### `PATCH /v1/schedules/{scheduleID}`

Enables, disables, or updates a schedule.

### `DELETE /v1/schedules/{scheduleID}`

Deletes a schedule.

## Status and HTTP semantics

| Status | Meaning |
| --- | --- |
| 200 | Successful read or update |
| 201 | Resource created |
| 202 | Task accepted for asynchronous execution |
| 204 | Successful delete with no response body |
| 400 | Malformed JSON or invalid request |
| 404 | Resource does not exist |
| 409 | State or uniqueness conflict |
| 422 | Validation failure |
| 500 | Unexpected server error |
| 503 | Service is not ready or a required dependency is unavailable |

## Task status lifecycle

```text
pending -> running -> completed
pending -> running -> failed
pending -> running -> cancelled
pending -> cancelled
```

Clients should poll `GET /v1/tasks/{taskID}` using bounded backoff. A future API version may expose server-sent events or webhooks without changing the task model.

## API implementation note

The agent, workflow, task, memory, and schedule routes are implemented in the current server composition. Tool and provider registration remain internal runtime concerns; they are intentionally not exposed as unrestricted public endpoints.
