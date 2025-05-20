package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler handles scheduling of backup tasks
type Scheduler struct {
	cron        *cron.Cron
	expression  string
	backupFunc  func() error
	entryID     cron.EntryID
	running     bool
	mutex       sync.Mutex
}

// New creates a new scheduler
func New(cronExpression string, backupFunc func() error) *Scheduler {
	// Create a new cron scheduler with seconds field enabled
	c := cron.New(cron.WithSeconds())

	return &Scheduler{
		cron:       c,
		expression: cronExpression,
		backupFunc: backupFunc,
		running:    false,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.running {
		return nil // Already running
	}

	// Add the backup function to the cron scheduler
	entryID, err := s.cron.AddFunc(s.expression, func() {
		fmt.Printf("Scheduled backup triggered at %s\n", time.Now().Format(time.RFC3339))
		
		// Execute the backup function
		if err := s.backupFunc(); err != nil {
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

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.running {
		return // Not running
	}

	// Remove the scheduled job
	s.cron.Remove(s.entryID)

	// Stop the cron scheduler
	s.cron.Stop()
	s.running = false
}

// RunNow executes a backup immediately
func (s *Scheduler) RunNow() error {
	fmt.Printf("Manual backup triggered at %s\n", time.Now().Format(time.RFC3339))
	
	// Execute the backup function
	if err := s.backupFunc(); err != nil {
		return fmt.Errorf("manual backup failed: %w", err)
	}

	fmt.Printf("Manual backup completed successfully at %s\n", time.Now().Format(time.RFC3339))
	return nil
}
