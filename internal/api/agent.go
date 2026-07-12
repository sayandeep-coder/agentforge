package api

import (
	"net/http"

	"github.com/agentforge/agentforge/internal/models"
	"github.com/agentforge/agentforge/internal/service"
	"github.com/gin-gonic/gin"
)

type agentHandler struct{ service service.AgentService }

type createAgentRequest struct {
	Name         string  `json:"name" binding:"required"`
	Description  string  `json:"description"`
	SystemPrompt string  `json:"system_prompt" binding:"required"`
	Model        string  `json:"model" binding:"required"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
}

type updateAgentRequest struct {
	Name         *string             `json:"name"`
	Description  *string             `json:"description"`
	SystemPrompt *string             `json:"system_prompt"`
	Model        *string             `json:"model"`
	Temperature  *float64            `json:"temperature"`
	MaxTokens    *int                `json:"max_tokens"`
	Status       *models.AgentStatus `json:"status"`
}

func newAgentHandler(agentService service.AgentService) *agentHandler {
	return &agentHandler{service: agentService}
}

func (h *agentHandler) create(c *gin.Context) {
	var request createAgentRequest
	if err := bindJSON(c, &request); err != nil {
		respondError(c, err)
		return
	}
	agent, err := h.service.Create(c.Request.Context(), service.CreateAgentInput{
		Name: request.Name, Description: request.Description, SystemPrompt: request.SystemPrompt,
		Model: request.Model, Temperature: request.Temperature, MaxTokens: request.MaxTokens,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusCreated, "agent created", agent)
}

func (h *agentHandler) list(c *gin.Context) {
	offset, limit, err := pagination(c)
	if err != nil {
		respondError(c, err)
		return
	}
	agents, total, err := h.service.List(c.Request.Context(), models.AgentStatus(c.Query("status")), offset, limit)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "agents listed", gin.H{"items": agents, "total": total, "offset": offset, "limit": limit})
}

func (h *agentHandler) get(c *gin.Context) {
	id, err := pathUUID(c, "agentID")
	if err != nil {
		respondError(c, err)
		return
	}
	agent, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "agent retrieved", agent)
}

func (h *agentHandler) update(c *gin.Context) {
	id, err := pathUUID(c, "agentID")
	if err != nil {
		respondError(c, err)
		return
	}
	var request updateAgentRequest
	if err := bindJSON(c, &request); err != nil {
		respondError(c, err)
		return
	}
	agent, err := h.service.Update(c.Request.Context(), id, service.UpdateAgentInput{
		Name: request.Name, Description: request.Description, SystemPrompt: request.SystemPrompt,
		Model: request.Model, Temperature: request.Temperature, MaxTokens: request.MaxTokens, Status: request.Status,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "agent updated", agent)
}

func (h *agentHandler) delete(c *gin.Context) {
	id, err := pathUUID(c, "agentID")
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
