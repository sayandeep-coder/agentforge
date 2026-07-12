// Package worker provides bounded concurrent task execution.
package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/agentforge/agentforge/internal/queue"
	"go.uber.org/zap"
)

var (
	// ErrAlreadyStarted indicates that a worker pool was started more than once.
	ErrAlreadyStarted = errors.New("worker: already started")
	// ErrNotStarted indicates that shutdown was requested before startup.
	ErrNotStarted = errors.New("worker: not started")
)

// Executor executes one queued task message.
type Executor interface {
	Execute(ctx context.Context, message queue.Message) error
}

// RetryPolicy controls retry count and delay between attempts.
type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
}

// Config configures a worker pool.
type Config struct {
	Workers  int
	Queue    queue.Queue
	Executor Executor
	Retry    RetryPolicy
	Logger   *zap.Logger
}

// Pool manages a fixed set of workers and their lifecycle.
type Pool struct {
	queue    queue.Queue
	executor Executor
	retry    RetryPolicy
	logger   *zap.Logger

	ctx    context.Context
	cancel context.CancelFunc
	group  sync.WaitGroup

	startOnce    sync.Once
	shutdownOnce sync.Once
	stateMu      sync.RWMutex
	started      bool
	errors       chan error
	done         chan struct{}
	shutdownErr  error
}

// NewPool validates configuration and creates a stopped worker pool.
func NewPool(config Config) (*Pool, error) {
	if config.Workers < 1 {
		return nil, fmt.Errorf("worker count must be positive: %d", config.Workers)
	}
	if config.Queue == nil {
		return nil, errors.New("worker queue is required")
	}
	if config.Executor == nil {
		return nil, errors.New("worker executor is required")
	}
	if config.Retry.MaxAttempts < 0 {
		return nil, errors.New("worker max attempts must not be negative")
	}
	if config.Retry.Backoff < 0 {
		return nil, errors.New("worker retry backoff must not be negative")
	}
	if config.Logger == nil {
		config.Logger = zap.NewNop()
	}
	return &Pool{
		queue:    config.Queue,
		executor: config.Executor,
		retry:    config.Retry,
		logger:   config.Logger,
		errors:   make(chan error, config.Workers*2),
	}, nil
}

// Start starts the configured worker goroutines.
func (p *Pool) Start(parent context.Context) error {
	started := false
	p.startOnce.Do(func() {
		p.ctx, p.cancel = context.WithCancel(parent)
		p.stateMu.Lock()
		p.started = true
		p.stateMu.Unlock()
		started = true
	})
	if !started {
		return ErrAlreadyStarted
	}

	workerCount := cap(p.errors) / 2
	p.group.Add(workerCount)
	for workerID := 0; workerID < workerCount; workerID++ {
		go p.run(workerID)
	}
	return nil
}

// Errors returns terminal task errors after retries are exhausted.
func (p *Pool) Errors() <-chan error { return p.errors }

// Shutdown cancels workers, closes the queue, and waits for all goroutines to exit.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.shutdownOnce.Do(func() {
		p.stateMu.RLock()
		started := p.started
		p.stateMu.RUnlock()
		if !started {
			p.shutdownErr = ErrNotStarted
			return
		}
		p.cancel()
		if err := p.queue.Close(); err != nil {
			p.shutdownErr = fmt.Errorf("close worker queue: %w", err)
			return
		}
		p.done = make(chan struct{})
		go func() {
			p.group.Wait()
			close(p.done)
			close(p.errors)
		}()
	})
	if p.shutdownErr != nil {
		return p.shutdownErr
	}
	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown worker pool: %w", ctx.Err())
	}
}

func (p *Pool) run(workerID int) {
	defer p.group.Done()
	for {
		message, err := p.queue.Dequeue(p.ctx)
		if err != nil {
			if !errors.Is(err, queue.ErrClosed) && !errors.Is(err, context.Canceled) {
				p.reportError(fmt.Errorf("worker %d dequeue: %w", workerID, err))
			}
			return
		}
		if err := p.execute(workerID, message); err != nil {
			p.reportError(err)
		}
	}
}

func (p *Pool) execute(workerID int, message queue.Message) error {
	err := p.executor.Execute(p.ctx, message)
	if err == nil {
		return nil
	}
	if message.Attempt >= p.retry.MaxAttempts {
		return fmt.Errorf("worker %d task %s failed after attempt %d: %w", workerID, message.TaskID, message.Attempt+1, err)
	}

	if p.retry.Backoff > 0 {
		timer := time.NewTimer(p.retry.Backoff)
		select {
		case <-p.ctx.Done():
			timer.Stop()
			return fmt.Errorf("worker %d retry cancelled for task %s: %w", workerID, message.TaskID, p.ctx.Err())
		case <-timer.C:
		}
	}

	message.Attempt++
	if err := p.queue.Enqueue(p.ctx, message); err != nil {
		return fmt.Errorf("worker %d requeue task %s: %w", workerID, message.TaskID, err)
	}
	p.logger.Warn("task execution failed; retrying",
		zap.Int("worker_id", workerID),
		zap.String("task_id", message.TaskID.String()),
		zap.Int("attempt", message.Attempt),
		zap.Error(err),
	)
	return nil
}

func (p *Pool) reportError(err error) {
	select {
	case p.errors <- err:
	case <-p.ctx.Done():
	}
}
