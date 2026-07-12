package api

import (
	"net/http"

	"github.com/agentforge/agentforge/internal/middleware"
	"github.com/agentforge/agentforge/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Dependencies contains application services required by HTTP routes.
type Dependencies struct {
	Agents    service.AgentService
	Workflows service.WorkflowService
	Tasks     service.TaskService
	Memory    service.MemoryService
	Schedules service.ScheduleService
	Logger    *zap.Logger
}

// NewRouter creates a configured Gin router with versioned REST routes.
func NewRouter(dependencies Dependencies) *gin.Engine {
	if dependencies.Logger == nil {
		dependencies.Logger = zap.NewNop()
	}
	router := gin.New()
	router.Use(gin.Recovery(), middleware.CORS(), middleware.RequestID(), middleware.AccessLog(dependencies.Logger))
	router.GET("/healthz", healthHandler)
	router.GET("/readyz", readyHandler)

	agentHandler := newAgentHandler(dependencies.Agents)
	workflowHandler := newWorkflowHandler(dependencies.Workflows)
	taskHandler := newTaskHandler(dependencies.Tasks)
	memoryHandler := newMemoryHandler(dependencies.Memory)
	scheduleHandler := newScheduleHandler(dependencies.Schedules)

	v1 := router.Group("/v1")
	v1.POST("/agents", agentHandler.create)
	v1.GET("/agents", agentHandler.list)
	v1.GET("/agents/:agentID", agentHandler.get)
	v1.PATCH("/agents/:agentID", agentHandler.update)
	v1.DELETE("/agents/:agentID", agentHandler.delete)
	v1.POST("/agents/:agentID/workflows", workflowHandler.create)
	v1.GET("/agents/:agentID/workflows", workflowHandler.listByAgent)
	v1.GET("/workflows/:workflowID", workflowHandler.get)
	v1.PATCH("/workflows/:workflowID", workflowHandler.update)
	v1.DELETE("/workflows/:workflowID", workflowHandler.delete)
	v1.POST("/tasks", taskHandler.submit)
	v1.GET("/tasks/:taskID", taskHandler.get)
	v1.POST("/tasks/:taskID/cancel", taskHandler.cancel)
	v1.GET("/agents/:agentID/memory/:key", memoryHandler.get)
	v1.PUT("/agents/:agentID/memory/:key", memoryHandler.upsert)
	v1.DELETE("/agents/:agentID/memory/:key", memoryHandler.delete)
	v1.POST("/agents/:agentID/schedules", scheduleHandler.create)
	v1.GET("/agents/:agentID/schedules", scheduleHandler.listByAgent)
	v1.PATCH("/schedules/:scheduleID", scheduleHandler.update)
	v1.DELETE("/schedules/:scheduleID", scheduleHandler.delete)
	return router
}

func healthHandler(c *gin.Context) {
	respond(c, http.StatusOK, "service is healthy", gin.H{"status": "ok"})
}

func readyHandler(c *gin.Context) {
	respond(c, http.StatusOK, "service is ready", gin.H{"status": "ready"})
}
