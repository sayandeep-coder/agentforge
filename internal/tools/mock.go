package tools

import (
	"context"
	"errors"
	"fmt"
	"strconv"
)

// MockTool returns a configured response and is useful for local development.
type MockTool struct {
	ToolName string
	Response any
}

// Name returns the configured tool name.
func (t MockTool) Name() string { return t.ToolName }

// Execute returns the configured mock response unless the context is cancelled.
func (t MockTool) Execute(ctx context.Context, _ any) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return t.Response, nil
	}
}

// Calculator evaluates a simple two-operand arithmetic request.
type Calculator struct{}

// Name returns the calculator tool name.
func (Calculator) Name() string { return "calculator" }

// Execute accepts CalculatorInput and returns CalculatorOutput.
func (Calculator) Execute(ctx context.Context, input any) (any, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	request, ok := input.(CalculatorInput)
	if !ok {
		return nil, errors.New("calculator input must be CalculatorInput")
	}
	var result float64
	switch request.Operator {
	case "+":
		result = request.Left + request.Right
	case "-":
		result = request.Left - request.Right
	case "*":
		result = request.Left * request.Right
	case "/":
		if request.Right == 0 {
			return nil, errors.New("calculator cannot divide by zero")
		}
		result = request.Left / request.Right
	default:
		return nil, fmt.Errorf("unsupported calculator operator: %s", request.Operator)
	}
	return CalculatorOutput{Result: result, Text: strconv.FormatFloat(result, 'f', -1, 64)}, nil
}

// CalculatorInput describes a basic arithmetic operation.
type CalculatorInput struct {
	Left     float64
	Operator string
	Right    float64
}

// CalculatorOutput contains numeric and string representations of a result.
type CalculatorOutput struct {
	Result float64
	Text   string
}
