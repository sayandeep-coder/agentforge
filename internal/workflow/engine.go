// Package workflow executes ordered workflow definitions.
package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/agentforge/agentforge/internal/llm"
	"github.com/agentforge/agentforge/internal/models"
	"github.com/agentforge/agentforge/internal/tools"
)

// Engine executes workflow steps using provider ports.
type Engine struct {
	tools tools.Registry
	llm   llm.LLMProvider
}

// NewEngine creates a workflow engine.
func NewEngine(toolRegistry tools.Registry, provider llm.LLMProvider) (*Engine, error) {
	if toolRegistry == nil {
		return nil, errors.New("workflow tool registry is required")
	}
	if provider == nil {
		return nil, errors.New("workflow LLM provider is required")
	}
	return &Engine{tools: toolRegistry, llm: provider}, nil
}

// Execute runs a workflow from its first ordered step and returns the final JSON result.
func (e *Engine) Execute(ctx context.Context, agent *models.Agent, definition *models.Workflow, input json.RawMessage) (json.RawMessage, error) {
	if agent == nil || definition == nil {
		return nil, errors.New("workflow agent and definition are required")
	}
	if !json.Valid(input) {
		return nil, errors.New("workflow input must be valid JSON")
	}
	current := input
	for _, step := range definition.Steps {
		var output any
		var err error
		switch strings.ToLower(step.StepType) {
		case "llm", "model":
			prompt := fmt.Sprintf("%s\n\nWorkflow step: %s\nInput: %s", agent.SystemPrompt, step.StepName, current)
			var response string
			response, err = e.llm.Generate(ctx, prompt)
			output = response
		case "tool":
			output, err = e.executeTool(ctx, step, current)
		default:
			err = fmt.Errorf("unsupported workflow step type: %s", step.StepType)
		}
		if err != nil {
			return nil, fmt.Errorf("execute workflow step %s: %w", step.StepName, err)
		}
		current, err = json.Marshal(output)
		if err != nil {
			return nil, fmt.Errorf("encode workflow step %s output: %w", step.StepName, err)
		}
	}
	return current, nil
}

type toolConfig struct {
	Name  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
}

func (e *Engine) executeTool(ctx context.Context, step models.WorkflowStep, current json.RawMessage) (any, error) {
	var config toolConfig
	if len(step.Config) > 0 && !json.Valid(step.Config) {
		return nil, errors.New("tool step config must be valid JSON")
	}
	if len(step.Config) > 0 {
		if err := json.Unmarshal(step.Config, &config); err != nil {
			return nil, fmt.Errorf("decode tool step config: %w", err)
		}
	}
	if config.Name == "" {
		return nil, errors.New("tool step requires config.tool")
	}
	toolInput := any(nil)
	if len(config.Input) > 0 {
		if err := json.Unmarshal(config.Input, &toolInput); err != nil {
			return nil, fmt.Errorf("decode tool step input: %w", err)
		}
	} else if len(current) > 0 {
		if err := json.Unmarshal(current, &toolInput); err != nil {
			return nil, fmt.Errorf("decode workflow input for tool: %w", err)
		}
	}
	return e.tools.Execute(ctx, config.Name, toolInput)
}
