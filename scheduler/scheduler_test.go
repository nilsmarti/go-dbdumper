package scheduler

import (
	"sync"
	"testing"
)

func TestScheduler(t *testing.T) {
	// Create a counter to track how many times the backup function is called
	var counter int
	var mu sync.Mutex

	// Create a test backup function
	backupFunc := func() error {
		mu.Lock()
		defer mu.Unlock()
		counter++
		return nil
	}

	// Create a scheduler with a cron expression that runs every minute
	s := New("* * * * *", backupFunc)

	// Start the scheduler
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// For testing purposes, manually trigger a backup instead of waiting for cron
	err = s.RunNow()
	if err != nil {
		t.Fatalf("Failed to run backup: %v", err)
	}

	// Stop the scheduler
	s.Stop()

	// Check that the backup function was called at least once
	mu.Lock()
	if counter < 1 {
		t.Errorf("Expected backup function to be called at least once, got %d", counter)
	}
	mu.Unlock()
}

func TestRunNow(t *testing.T) {
	// Create a counter to track how many times the backup function is called
	var counter int

	// Create a test backup function
	backupFunc := func() error {
		counter++
		return nil
	}

	// Create a scheduler with a cron expression that never runs
	s := New("0 0 31 2 *", backupFunc) // February 31st (never happens)

	// Run the backup immediately
	err := s.RunNow()
	if err != nil {
		t.Fatalf("Failed to run backup: %v", err)
	}

	// Check that the backup function was called exactly once
	if counter != 1 {
		t.Errorf("Expected backup function to be called exactly once, got %d", counter)
	}
}
