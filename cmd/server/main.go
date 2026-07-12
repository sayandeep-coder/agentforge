package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agentforge/agentforge/internal/agent"
	"github.com/agentforge/agentforge/internal/api"
	"github.com/agentforge/agentforge/internal/config"
	"github.com/agentforge/agentforge/internal/database"
	"github.com/agentforge/agentforge/internal/llm"
	"github.com/agentforge/agentforge/internal/logger"
	"github.com/agentforge/agentforge/internal/queue"
	"github.com/agentforge/agentforge/internal/repository"
	"github.com/agentforge/agentforge/internal/service"
	"github.com/agentforge/agentforge/internal/tools"
	"github.com/agentforge/agentforge/internal/worker"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic("load configuration: " + err.Error())
	}

	log, err := logger.New(cfg.App.Environment, cfg.Log.Level)
	if err != nil {
		panic("create logger: " + err.Error())
	}
	defer func() { _ = log.Sync() }()

	databaseConnection, err := database.Open(context.Background(), cfg.Database)
	if err != nil {
		log.Fatal("open database", zap.Error(err))
	}
	defer func() { _ = databaseConnection.Close() }()

	agentRepository := repository.NewAgentRepository(databaseConnection.DB)
	workflowRepository := repository.NewWorkflowRepository(databaseConnection.DB)
	taskRepository := repository.NewTaskRepository(databaseConnection.DB)
	memoryRepository := repository.NewMemoryRepository(databaseConnection.DB)
	scheduleRepository := repository.NewScheduleRepository(databaseConnection.DB)
	taskQueue, err := queue.NewMemoryQueue(queue.MemoryConfig{Capacity: cfg.Worker.QueueCapacity})
	if err != nil {
		log.Fatal("create task queue", zap.Error(err))
	}
	toolRegistry := tools.NewRegistry()
	if err := toolRegistry.Register(tools.Calculator{}); err != nil {
		log.Fatal("register calculator tool", zap.Error(err))
	}
	if err := toolRegistry.Register(tools.MockTool{ToolName: "web_search", Response: map[string]string{"status": "mock"}}); err != nil {
		log.Fatal("register web search tool", zap.Error(err))
	}
	runtime, err := agent.NewRuntime(agent.Config{
		Agents:       agentRepository,
		Workflows:    workflowRepository,
		Tasks:        taskRepository,
		ToolRegistry: toolRegistry,
		LLMProvider:  llm.MockProvider{Response: "mock LLM response"},
		Logger:       log,
	})
	if err != nil {
		log.Fatal("create agent runtime", zap.Error(err))
	}
	workerPool, err := worker.NewPool(worker.Config{
		Workers:  cfg.Worker.Count,
		Queue:    taskQueue,
		Executor: runtime,
		Retry:    worker.RetryPolicy{MaxAttempts: 2, Backoff: time.Second},
		Logger:   log,
	})
	if err != nil {
		log.Fatal("create worker pool", zap.Error(err))
	}
	if err := workerPool.Start(context.Background()); err != nil {
		log.Fatal("start worker pool", zap.Error(err))
	}
	go logWorkerErrors(workerPool, log)

	router := api.NewRouter(api.Dependencies{
		Agents:    service.NewAgentService(agentRepository),
		Workflows: service.NewWorkflowService(workflowRepository),
		Tasks:     service.NewTaskService(taskRepository, queue.NewTaskPublisher(taskQueue)),
		Memory:    service.NewMemoryService(memoryRepository),
		Schedules: service.NewScheduleService(scheduleRepository),
		Logger:    log,
	})
	server := newHTTPServer(cfg, router)
	serverErr := make(chan error, 1)
	go func() {
		log.Info("starting HTTP server", zap.String("address", cfg.Address()))
		serverErr <- server.ListenAndServe()
	}()

	shutdownSignal := make(chan os.Signal, 1)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(shutdownSignal)

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP server stopped unexpectedly", zap.Error(err))
			shutdownWorkerPool(workerPool, cfg.Server.ShutdownTimeout, log)
			os.Exit(1)
		}
	case signalValue := <-shutdownSignal:
		log.Info("shutdown signal received", zap.String("signal", signalValue.String()))
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Error("HTTP server shutdown failed", zap.Error(err))
			os.Exit(1)
		}
		shutdownWorkerPool(workerPool, cfg.Server.ShutdownTimeout, log)
	}
}

func logWorkerErrors(pool *worker.Pool, log *zap.Logger) {
	for err := range pool.Errors() {
		log.Error("worker task execution failed", zap.Error(err))
	}
}

func shutdownWorkerPool(pool *worker.Pool, timeout time.Duration, log *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := pool.Shutdown(ctx); err != nil {
		log.Error("worker pool shutdown failed", zap.Error(err))
	}
}

func newHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.Address(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
