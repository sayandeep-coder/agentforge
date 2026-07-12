package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func TestSchedulerTriggersAndStops(t *testing.T) {
	var calls atomic.Int32
	scheduleID := uuid.New()
	scheduler, err := New(func(_ context.Context, id uuid.UUID) error {
		if id == scheduleID {
			calls.Add(1)
		}
		return nil
	}, zap.NewNop())
	if err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Add(scheduleID, "*/1 * * * * *"); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(2 * time.Second)
	for calls.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for schedule trigger")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	stopContext, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := scheduler.Stop(stopContext); err != nil {
		t.Fatal(err)
	}
}
