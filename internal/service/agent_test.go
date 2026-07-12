package service

import (
	"context"
	"testing"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/google/uuid"
)

type fakeAgentRepository struct {
	created *models.Agent
	agent   *models.Agent
}

func (r *fakeAgentRepository) Create(_ context.Context, agent *models.Agent) error {
	r.created = agent
	return nil
}

func (r *fakeAgentRepository) GetByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return r.agent, nil
}

func (r *fakeAgentRepository) List(_ context.Context, _ models.AgentStatus, _, _ int) ([]models.Agent, int64, error) {
	return nil, 0, nil
}

func (r *fakeAgentRepository) Update(_ context.Context, agent *models.Agent) error {
	r.agent = agent
	return nil
}

func (r *fakeAgentRepository) Delete(_ context.Context, _ uuid.UUID) error { return nil }

func TestAgentServiceCreateAppliesDefaults(t *testing.T) {
	repository := &fakeAgentRepository{}
	service := NewAgentService(repository)

	agent, err := service.Create(context.Background(), CreateAgentInput{
		Name:         "  planner ",
		SystemPrompt: " plan carefully ",
		Model:        "mock",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if agent.ID == uuid.Nil || repository.created != agent {
		t.Fatal("Create() did not persist the created agent")
	}
	if agent.Name != "planner" || agent.Temperature != defaultAgentTemperature || agent.MaxTokens != defaultAgentMaxTokens {
		t.Fatalf("Create() returned unexpected agent: %+v", agent)
	}
}

func TestAgentServiceRejectsInvalidCreate(t *testing.T) {
	service := NewAgentService(&fakeAgentRepository{})
	_, err := service.Create(context.Background(), CreateAgentInput{Name: "missing fields"})
	if err == nil || !containsError(err, ErrInvalidInput) {
		t.Fatalf("Create() error = %v, want ErrInvalidInput", err)
	}
}

func containsError(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		unwrapped, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapped.Unwrap()
	}
	return false
}
