package api

import (
	"encoding/json"
	"net/http"

	"github.com/agentforge/agentforge/internal/service"
	"github.com/gin-gonic/gin"
)

type memoryHandler struct{ service service.MemoryService }

type upsertMemoryRequest struct {
	Value json.RawMessage `json:"value" binding:"required"`
}

func newMemoryHandler(memoryService service.MemoryService) *memoryHandler {
	return &memoryHandler{service: memoryService}
}

func (h *memoryHandler) get(c *gin.Context) {
	agentID, err := pathUUID(c, "agentID")
	if err != nil {
		respondError(c, err)
		return
	}
	memory, err := h.service.Get(c.Request.Context(), agentID, c.Param("key"))
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "memory retrieved", memory)
}

func (h *memoryHandler) upsert(c *gin.Context) {
	agentID, err := pathUUID(c, "agentID")
	if err != nil {
		respondError(c, err)
		return
	}
	var request upsertMemoryRequest
	if err := bindJSON(c, &request); err != nil {
		respondError(c, err)
		return
	}
	memory, err := h.service.Upsert(c.Request.Context(), agentID, c.Param("key"), request.Value)
	if err != nil {
		respondError(c, err)
		return
	}
	respond(c, http.StatusOK, "memory saved", memory)
}

func (h *memoryHandler) delete(c *gin.Context) {
	agentID, err := pathUUID(c, "agentID")
	if err != nil {
		respondError(c, err)
		return
	}
	if err := h.service.Delete(c.Request.Context(), agentID, c.Param("key")); err != nil {
		respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
