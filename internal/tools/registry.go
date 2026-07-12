// Package tools provides a concurrency-safe registry for agent tools.
package tools

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	// ErrNotFound indicates that a named tool is not registered.
	ErrNotFound = errors.New("tools: not found")
	// ErrDuplicate indicates that a tool name is already registered.
	ErrDuplicate = errors.New("tools: duplicate registration")
)

// Tool is the execution contract implemented by every agent tool.
type Tool interface {
	Name() string
	Execute(ctx context.Context, input any) (any, error)
}

// Registry resolves and executes registered tools.
type Registry interface {
	Register(tool Tool) error
	Get(name string) (Tool, error)
	Execute(ctx context.Context, name string, input any) (any, error)
	Names() []string
}

type registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() Registry {
	return &registry{tools: make(map[string]Tool)}
}

func (r *registry) Register(tool Tool) error {
	if tool == nil {
		return errors.New("tool is required")
	}
	name := strings.TrimSpace(tool.Name())
	if name == "" {
		return errors.New("tool name is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("%w: %s", ErrDuplicate, name)
	}
	r.tools[name] = tool
	return nil
}

func (r *registry) Get(name string) (Tool, error) {
	name = strings.TrimSpace(name)
	r.mu.RLock()
	tool, exists := r.tools[name]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return tool, nil
}

func (r *registry) Execute(ctx context.Context, name string, input any) (any, error) {
	tool, err := r.Get(name)
	if err != nil {
		return nil, err
	}
	result, err := tool.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("execute tool %s: %w", name, err)
	}
	return result, nil
}

func (r *registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
