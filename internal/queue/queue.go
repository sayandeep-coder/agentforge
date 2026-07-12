// Package queue defines the task dispatch port and its in-memory adapter.
package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	// ErrClosed indicates that the queue no longer accepts or serves messages.
	ErrClosed = errors.New("queue: closed")
)

// Message is the durable task reference exchanged between producers and workers.
type Message struct {
	TaskID     uuid.UUID
	Attempt    int
	EnqueuedAt time.Time
}

// Queue is the transport-independent task dispatch contract.
type Queue interface {
	Enqueue(ctx context.Context, message Message) error
	Dequeue(ctx context.Context) (Message, error)
	Close() error
}

// MemoryConfig configures the in-memory queue adapter.
type MemoryConfig struct {
	Capacity int
}

// MemoryQueue is a bounded, process-local FIFO queue suitable for development and tests.
type MemoryQueue struct {
	items    chan Message
	closed   chan struct{}
	closeErr error
	closeMu  sync.RWMutex
	closeOne sync.Once
}

// NewMemoryQueue creates a bounded in-memory queue.
func NewMemoryQueue(config MemoryConfig) (*MemoryQueue, error) {
	if config.Capacity < 1 {
		return nil, fmt.Errorf("memory queue capacity must be positive: %d", config.Capacity)
	}
	return &MemoryQueue{
		items:  make(chan Message, config.Capacity),
		closed: make(chan struct{}),
	}, nil
}

// Enqueue waits for capacity or context cancellation, then publishes a message.
func (q *MemoryQueue) Enqueue(ctx context.Context, message Message) error {
	if message.TaskID == uuid.Nil {
		return errors.New("queue: task ID is required")
	}
	if message.EnqueuedAt.IsZero() {
		message.EnqueuedAt = time.Now().UTC()
	}
	if q.isClosed() {
		return ErrClosed
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("enqueue message: %w", ctx.Err())
	case <-q.closed:
		return ErrClosed
	case q.items <- message:
		return nil
	}
}

// Dequeue waits for a message or context cancellation.
func (q *MemoryQueue) Dequeue(ctx context.Context) (Message, error) {
	select {
	case <-ctx.Done():
		return Message{}, fmt.Errorf("dequeue message: %w", ctx.Err())
	case message := <-q.items:
		return message, nil
	case <-q.closed:
		return Message{}, ErrClosed
	}
}

// Close prevents new messages and wakes blocked producers and consumers.
func (q *MemoryQueue) Close() error {
	q.closeOne.Do(func() {
		q.closeMu.Lock()
		q.closeErr = nil
		q.closeMu.Unlock()
		close(q.closed)
	})
	q.closeMu.RLock()
	defer q.closeMu.RUnlock()
	return q.closeErr
}

func (q *MemoryQueue) isClosed() bool {
	select {
	case <-q.closed:
		return true
	default:
		return false
	}
}
