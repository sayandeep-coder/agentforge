package tools

import (
	"context"
	"errors"
	"testing"
)

func TestRegistryRegistersAndExecutes(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(MockTool{ToolName: "weather", Response: map[string]string{"status": "mock"}}); err != nil {
		t.Fatal(err)
	}
	result, err := registry.Execute(context.Background(), "weather", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.(map[string]string)["status"] != "mock" {
		t.Fatalf("unexpected mock result: %#v", result)
	}
	if err := registry.Register(MockTool{ToolName: "weather"}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestCalculator(t *testing.T) {
	result, err := (Calculator{}).Execute(context.Background(), CalculatorInput{Left: 2, Operator: "*", Right: 3})
	if err != nil {
		t.Fatal(err)
	}
	if result.(CalculatorOutput).Result != 6 {
		t.Fatalf("result = %#v", result)
	}
}
