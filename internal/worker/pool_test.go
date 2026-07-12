package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/agentforge/agentforge/internal/queue"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type fakeExecutor struct {
	mu       sync.Mutex
	attempts map[uuid.UUID]int
	failures int
}

func (e *fakeExecutor) Execute(_ context.Context, message queue.Message) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.attempts[message.TaskID]++
	if e.attempts[message.TaskID] <= e.failures {
		return errors.New("transient failure")
	}
	return nil
}

func TestPoolRetriesAndExecutes(t *testing.T) {
	taskQueue, err := queue.NewMemoryQueue(queue.MemoryConfig{Capacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{attempts: make(map[uuid.UUID]int), failures: 2}
	pool, err := NewPool(Config{
		Workers:  1,
		Queue:    taskQueue,
		Executor: executor,
		Retry:    RetryPolicy{MaxAttempts: 2},
		Logger:   zap.NewNop(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	taskID := uuid.New()
	if err := taskQueue.Enqueue(context.Background(), queue.Message{TaskID: taskID}); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(2 * time.Second)
	for {
		executor.mu.Lock()
		attempts := executor.attempts[taskID]
		executor.mu.Unlock()
		if attempts == 3 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("executor attempts = %d, want 3", attempts)
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := pool.Shutdown(shutdownContext); err != nil {
		t.Fatal(err)
	}
}

func TestPoolReportsTerminalError(t *testing.T) {
	taskQueue, err := queue.NewMemoryQueue(queue.MemoryConfig{Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{attempts: make(map[uuid.UUID]int), failures: 10}
	pool, err := NewPool(Config{Workers: 1, Queue: taskQueue, Executor: executor, Logger: zap.NewNop()})
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	taskID := uuid.New()
	if err := taskQueue.Enqueue(context.Background(), queue.Message{TaskID: taskID}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-pool.Errors():
		if err == nil {
			t.Fatal("received nil terminal error")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for terminal error")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := pool.Shutdown(shutdownContext); err != nil {
		t.Fatal(err)
	}
}
