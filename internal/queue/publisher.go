package queue

import (
	"context"

	"github.com/google/uuid"
)

// TaskPublisher adapts a queue to the application service submission port.
type TaskPublisher struct {
	queue Queue
}

// NewTaskPublisher creates a task publisher backed by the supplied queue.
func NewTaskPublisher(taskQueue Queue) *TaskPublisher {
	return &TaskPublisher{queue: taskQueue}
}

// Publish places a task reference on the queue.
func (p *TaskPublisher) Publish(ctx context.Context, taskID uuid.UUID) error {
	return p.queue.Enqueue(ctx, Message{TaskID: taskID})
}
