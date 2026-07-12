package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/agentforge/agentforge/internal/llm"
	"github.com/agentforge/agentforge/internal/models"
	"github.com/agentforge/agentforge/internal/tools"
)

func TestEngineExecutesOrderedSteps(t *testing.T) {
	registry := tools.NewRegistry()
	if err := registry.Register(tools.MockTool{ToolName: "mock_tool", Response: map[string]string{"ok": "true"}}); err != nil {
		t.Fatal(err)
	}
	engine, err := NewEngine(registry, llm.MockProvider{Response: "summary"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := engine.Execute(context.Background(), &models.Agent{SystemPrompt: "be concise"}, &models.Workflow{Steps: []models.WorkflowStep{
		{StepName: "lookup", StepType: "tool", Config: []byte(`{"tool":"mock_tool"}`), StepOrder: 1},
		{StepName: "summarize", StepType: "llm", StepOrder: 2},
	}}, json.RawMessage(`{"query":"agent runtime"}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != `"summary"` {
		t.Fatalf("result = %s", result)
	}
}
