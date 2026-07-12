package api

import (
	"encoding/json"
	"net/http"

	"github.com/agentforge/agentforge/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type taskHandler struct{ service service.TaskService }

type submitTaskRequest struct {
	AgentID    string          `json:"agent_id" binding:"required"`
	WorkflowID *string         `json:"workflow_id"`
	Priority   int             `json:"priority"`
	Input      json.RawMessage `json:"input" binding:"required"`
}

func newTaskHandler(taskService service.TaskService) *taskHandler {
	return &taskHandler{service: taskService}
}

func (h *taskHandler) submit(c *gin.Context) {
	var request submitTaskRequest
	if err := bindJSON(c, &request); err != nil {
		respondError(c, err)
		return
	}
	agentID, err := parseRequestUUID(request.AgentID, "agent_id")
	if err != nil {
		respondError(c, err)
		return
	}
	var workflowIDPtr *uuid.UUID
	if request.WorkflowID != nil {
		workflowID, parseErr := parseRequestUUID(*request.WorkflowID, "workflow_id")
		if parseErr != nil {
			respondError(c, parseErr)
			return
		}
		workflowIDPtr = &workflowID
	}
	task, err := h.service.Submit(c.Request.Context(), service.SubmitTaskInput{AgentID: agentID, WorkflowID: workflowIDPtr, Priority: request.Priority, Input: request.Input})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusAccepted, "task accepted", task)
}

func (h *taskHandler) get(c *gin.Context) {
	id, err := pathUUID(c, "taskID")
	if err != nil {
		respondError(c, err)
		return
	}
	task, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "task retrieved", task)
}

func (h *taskHandler) cancel(c *gin.Context) {
	id, err := pathUUID(c, "taskID")
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.service.Cancel(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusAccepted, "task cancellation requested", nil)
}
