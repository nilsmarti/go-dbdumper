package scheduler

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler handles scheduling of backup tasks
type Scheduler struct {
	cron       *cron.Cron
	expression string
	backupFunc func(context.Context) error
	entryID    cron.EntryID
	running    bool
	mutex      sync.Mutex
	ctx        context.Context
}

// New creates a new scheduler. The backup function receives the context passed
// to Start, so cancelling that context aborts in-flight backups triggered by
// the cron schedule.
func New(cronExpression string, backupFunc func(context.Context) error) *Scheduler {
	// Create a new cron scheduler with standard cron format (5 fields)
	c := cron.New()

	return &Scheduler{
		cron:       c,
		expression: cronExpression,
		backupFunc: backupFunc,
		running:    false,
	}
}

// Start starts the scheduler. The provided context is propagated to backup
// runs triggered by the cron schedule; cancelling it aborts any in-flight
// backup.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return nil // Already running
	}

	s.ctx = ctx

	// Add the backup function to the cron scheduler
	entryID, err := s.cron.AddFunc(s.expression, func() {
		fmt.Printf("Scheduled backup triggered at %s\n", time.Now().Format(time.RFC3339))

		// Execute the backup function
		if err := s.backupFunc(s.ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Printf("Scheduled backup cancelled at %s\n", time.Now().Format(time.RFC3339))
				return
			}
			fmt.Printf("Scheduled backup failed: %v\n", err)
		} else {
			fmt.Printf("Scheduled backup completed successfully at %s\n", time.Now().Format(time.RFC3339))
		}
	})

	if err != nil {
		return fmt.Errorf("failed to schedule backup: %w", err)
	}

	// Start the cron scheduler
	s.cron.Start()
	s.entryID = entryID
	s.running = true

	return nil
}

// Stop stops the scheduler. It does not cancel in-flight backups; cancel the
// context passed to Start for that.
func (s *Scheduler) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return // Not running
	}

	// Remove the scheduled job
	s.cron.Remove(s.entryID)

	// Stop the cron scheduler. cron.Stop returns a context that is done when
	// in-flight jobs complete; we wait for it so shutdown is orderly.
	<-s.cron.Stop().Done()

	s.running = false
}

// RunNow executes a backup immediately using the provided context.
func (s *Scheduler) RunNow(ctx context.Context) error {
	fmt.Printf("Manual backup triggered at %s\n", time.Now().Format(time.RFC3339))

	// Execute the backup function
	if err := s.backupFunc(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return fmt.Errorf("manual backup cancelled: %w", err)
		}
		return fmt.Errorf("manual backup failed: %w", err)
	}

	fmt.Printf("Manual backup completed successfully at %s\n", time.Now().Format(time.RFC3339))
	return nil
}
