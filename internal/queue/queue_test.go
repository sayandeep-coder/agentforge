package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMemoryQueueRoundTrip(t *testing.T) {
	queue, err := NewMemoryQueue(MemoryConfig{Capacity: 1})
	if err != nil {
		t.Fatalf("NewMemoryQueue() error = %v", err)
	}
	defer func() { _ = queue.Close() }()

	want := Message{TaskID: uuid.New(), Attempt: 1}
	if err := queue.Enqueue(context.Background(), want); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}
	got, err := queue.Dequeue(context.Background())
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}
	if got.TaskID != want.TaskID || got.Attempt != want.Attempt || got.EnqueuedAt.IsZero() {
		t.Fatalf("Dequeue() = %+v, want task %s attempt %d", got, want.TaskID, want.Attempt)
	}
}

func TestMemoryQueueClose(t *testing.T) {
	queue, err := NewMemoryQueue(MemoryConfig{Capacity: 1})
	if err != nil {
		t.Fatalf("NewMemoryQueue() error = %v", err)
	}
	if err := queue.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := queue.Enqueue(context.Background(), Message{TaskID: uuid.New()}); !errors.Is(err, ErrClosed) {
		t.Fatalf("Enqueue() error = %v, want ErrClosed", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := queue.Dequeue(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("Dequeue() error = %v, want ErrClosed", err)
	}
}
