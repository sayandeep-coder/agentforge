package llm

import (
	"context"
	"testing"
)

func TestMockProvider(t *testing.T) {
	provider := MockProvider{Response: "mock response"}
	got, err := provider.Generate(context.Background(), "prompt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "mock response" {
		t.Fatalf("response = %q", got)
	}
}
