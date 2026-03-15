package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/omaklabs/base/internal/testutil"
)

func TestSchedulerEnqueuesJobs(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	q := NewQueue(queries)
	s := NewScheduler(q)

	s.Add(Schedule{
		Name:     "test_recurring",
		Interval: 100 * time.Millisecond,
		JobType:  "recurring_job",
		Payload:  map[string]string{"action": "tick"},
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()

	// Wait 350ms — expect immediate + ~3 ticks = ~4 jobs.
	time.Sleep(350 * time.Millisecond)
	cancel()
	<-done

	jobs, err := q.List(context.Background(), "", 100, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// We expect at least 3 jobs (immediate + 2 ticks minimum in 350ms with 100ms interval).
	if len(jobs) < 3 {
		t.Errorf("expected at least 3 enqueued jobs, got %d", len(jobs))
	}

	for _, job := range jobs {
		if job.Type != "recurring_job" {
			t.Errorf("job.Type = %q, want %q", job.Type, "recurring_job")
		}
	}
}

func TestSchedulerRunsImmediately(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	q := NewQueue(queries)
	s := NewScheduler(q)

	s.Add(Schedule{
		Name:     "immediate_test",
		Interval: 10 * time.Second, // very long interval
		JobType:  "immediate_job",
		Payload:  nil,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()

	// Wait just 50ms — the job should be enqueued immediately, well before
	// the 10-second interval fires.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	jobs, err := q.List(context.Background(), "", 100, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	if len(jobs) < 1 {
		t.Fatal("expected at least 1 job to be enqueued immediately")
	}
	if jobs[0].Type != "immediate_job" {
		t.Errorf("job.Type = %q, want %q", jobs[0].Type, "immediate_job")
	}
}

func TestSchedulerStopsOnCancel(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	q := NewQueue(queries)
	s := NewScheduler(q)

	s.Add(Schedule{
		Name:     "stop_test",
		Interval: 50 * time.Millisecond,
		JobType:  "stop_job",
		Payload:  nil,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()

	// Let it run briefly, then cancel.
	time.Sleep(80 * time.Millisecond)
	cancel()
	<-done

	// Record the count after cancellation.
	jobsBefore, err := q.List(context.Background(), "", 1000, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	countBefore := len(jobsBefore)

	// Wait a bit more — no new jobs should appear.
	time.Sleep(200 * time.Millisecond)

	jobsAfter, err := q.List(context.Background(), "", 1000, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	countAfter := len(jobsAfter)

	if countAfter != countBefore {
		t.Errorf("expected no new jobs after cancel: before=%d, after=%d", countBefore, countAfter)
	}
}

func TestSchedulerMultipleSchedules(t *testing.T) {
	_, queries, cleanup := testutil.SetupTestDB(t)
	defer cleanup()

	q := NewQueue(queries)
	s := NewScheduler(q)

	s.Add(Schedule{
		Name:     "fast_schedule",
		Interval: 100 * time.Millisecond,
		JobType:  "fast_job",
		Payload:  nil,
	})
	s.Add(Schedule{
		Name:     "slow_schedule",
		Interval: 200 * time.Millisecond,
		JobType:  "slow_job",
		Payload:  nil,
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Start(ctx)
		close(done)
	}()

	time.Sleep(450 * time.Millisecond)
	cancel()
	<-done

	jobs, err := q.List(context.Background(), "", 100, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	fastCount := 0
	slowCount := 0
	for _, job := range jobs {
		switch job.Type {
		case "fast_job":
			fastCount++
		case "slow_job":
			slowCount++
		}
	}

	// fast_job: immediate + ~4 ticks in 450ms = ~5
	if fastCount < 3 {
		t.Errorf("expected at least 3 fast_job enqueues, got %d", fastCount)
	}
	// slow_job: immediate + ~2 ticks in 450ms = ~3
	if slowCount < 2 {
		t.Errorf("expected at least 2 slow_job enqueues, got %d", slowCount)
	}
	// fast should have more enqueues than slow.
	if fastCount <= slowCount {
		t.Errorf("expected fast_job (%d) to have more enqueues than slow_job (%d)", fastCount, slowCount)
	}
}

func TestEveryConstructors(t *testing.T) {
	tests := []struct {
		name     string
		schedule Schedule
		wantInt  time.Duration
	}{
		{
			name:     "Every",
			schedule: Every(30*time.Second, "every_job", nil),
			wantInt:  30 * time.Second,
		},
		{
			name:     "EveryMinutes",
			schedule: EveryMinutes(5, "minute_job", nil),
			wantInt:  5 * time.Minute,
		},
		{
			name:     "EveryHours",
			schedule: EveryHours(2, "hour_job", nil),
			wantInt:  2 * time.Hour,
		},
		{
			name:     "Daily",
			schedule: Daily("daily_job", nil),
			wantInt:  24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.schedule.Interval != tt.wantInt {
				t.Errorf("Interval = %v, want %v", tt.schedule.Interval, tt.wantInt)
			}
			if tt.schedule.JobType == "" {
				t.Error("JobType should not be empty")
			}
		})
	}
}
