package api

import (
	"encoding/json"
	"net/http"

	"github.com/agentforge/agentforge/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type workflowHandler struct{ service service.WorkflowService }

type createWorkflowRequest struct {
	Name        string                `json:"name" binding:"required"`
	Description string                `json:"description"`
	Steps       []workflowStepRequest `json:"steps" binding:"required,min=1"`
}

type workflowStepRequest struct {
	StepName   string          `json:"step_name" binding:"required"`
	StepType   string          `json:"step_type" binding:"required"`
	StepOrder  int             `json:"step_order" binding:"required"`
	ToolID     *uuid.UUID      `json:"tool_id"`
	NextStepID *uuid.UUID      `json:"next_step_id"`
	Config     json.RawMessage `json:"config"`
}

type updateWorkflowRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

func newWorkflowHandler(workflowService service.WorkflowService) *workflowHandler {
	return &workflowHandler{service: workflowService}
}

func (h *workflowHandler) create(c *gin.Context) {
	agentID, err := pathUUID(c, "agentID")
	if err != nil {
		respondError(c, err)
		return
	}
	var request createWorkflowRequest
	if err := bindJSON(c, &request); err != nil {
		respondError(c, err)
		return
	}
	steps := make([]service.WorkflowStepInput, 0, len(request.Steps))
	for _, step := range request.Steps {
		steps = append(steps, service.WorkflowStepInput{
			StepName:   step.StepName,
			StepType:   step.StepType,
			StepOrder:  step.StepOrder,
			ToolID:     step.ToolID,
			NextStepID: step.NextStepID,
			Config:     step.Config,
		})
	}
	workflow, err := h.service.Create(c.Request.Context(), service.CreateWorkflowInput{AgentID: agentID, Name: request.Name, Description: request.Description, Steps: steps})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusCreated, "workflow created", workflow)
}

func (h *workflowHandler) listByAgent(c *gin.Context) {
	agentID, err := pathUUID(c, "agentID")
	if err != nil {
		respondError(c, err)
		return
	}
	offset, limit, err := pagination(c)
	if err != nil {
		respondError(c, err)
		return
	}
	workflows, total, err := h.service.ListByAgent(c.Request.Context(), agentID, offset, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "workflows listed", gin.H{"items": workflows, "total": total, "offset": offset, "limit": limit})
}

func (h *workflowHandler) get(c *gin.Context) {
	id, err := pathUUID(c, "workflowID")
	if err != nil {
		respondError(c, err)
		return
	}
	workflow, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "workflow retrieved", workflow)
}

func (h *workflowHandler) update(c *gin.Context) {
	id, err := pathUUID(c, "workflowID")
	if err != nil {
		respondError(c, err)
		return
	}
	var request updateWorkflowRequest
	if err := bindJSON(c, &request); err != nil {
		respondError(c, err)
		return
	}
	workflow, err := h.service.Update(c.Request.Context(), id, service.UpdateWorkflowInput{
		Name: request.Name, Description: request.Description, IsActive: request.IsActive,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "workflow updated", workflow)
}

func (h *workflowHandler) delete(c *gin.Context) {
	id, err := pathUUID(c, "workflowID")
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
