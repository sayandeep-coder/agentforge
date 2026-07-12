// Package llm defines the model-provider port used by agent execution.
package llm

import (
	"context"
	"errors"
)

// LLMProvider generates model output for a prompt.
type LLMProvider interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

// MockProvider is a deterministic provider for local development and tests.
type MockProvider struct {
	Response string
}

// Generate returns the configured response while honoring context cancellation.
func (p MockProvider) Generate(ctx context.Context, _ string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		if p.Response == "" {
			return "", errors.New("mock LLM response is empty")
		}
		return p.Response, nil
	}
}
