package api

import (
	"net/http"

	"github.com/agentforge/agentforge/internal/service"
	"github.com/gin-gonic/gin"
)

type scheduleHandler struct{ service service.ScheduleService }

type createScheduleRequest struct {
	WorkflowID     *string `json:"workflow_id"`
	CronExpression string  `json:"cron_expression" binding:"required"`
	Enabled        *bool   `json:"enabled"`
}

type updateScheduleRequest struct {
	WorkflowID     *string `json:"workflow_id"`
	CronExpression *string `json:"cron_expression"`
	Enabled        *bool   `json:"enabled"`
}

func newScheduleHandler(scheduleService service.ScheduleService) *scheduleHandler {
	return &scheduleHandler{service: scheduleService}
}

func (h *scheduleHandler) create(c *gin.Context) {
	agentID, err := pathUUID(c, "agentID")
	if err != nil {
		respondError(c, err)
		return
	}
	var request createScheduleRequest
	if err := bindJSON(c, &request); err != nil {
		respondError(c, err)
		return
	}
	workflowID, err := optionalUUID(request.WorkflowID, "workflow_id")
	if err != nil {
		respondError(c, err)
		return
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	schedule, err := h.service.Create(c.Request.Context(), service.CreateScheduleInput{AgentID: agentID, WorkflowID: workflowID, CronExpression: request.CronExpression, Enabled: enabled})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusCreated, "schedule created", schedule)
}

func (h *scheduleHandler) listByAgent(c *gin.Context) {
	agentID, err := pathUUID(c, "agentID")
	if err != nil {
		respondError(c, err)
		return
	}
	schedules, err := h.service.ListByAgent(c.Request.Context(), agentID)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "schedules listed", schedules)
}

func (h *scheduleHandler) update(c *gin.Context) {
	scheduleID, err := pathUUID(c, "scheduleID")
	if err != nil {
		respondError(c, err)
		return
	}
	var request updateScheduleRequest
	if err := bindJSON(c, &request); err != nil {
		respondError(c, err)
		return
	}
	workflowID, err := optionalUUID(request.WorkflowID, "workflow_id")
	if err != nil {
		respondError(c, err)
		return
	}
	schedule, err := h.service.Update(c.Request.Context(), scheduleID, service.UpdateScheduleInput{WorkflowID: workflowID, CronExpression: request.CronExpression, Enabled: request.Enabled})
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "schedule updated", schedule)
}

func (h *scheduleHandler) delete(c *gin.Context) {
	scheduleID, err := pathUUID(c, "scheduleID")
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), scheduleID); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
