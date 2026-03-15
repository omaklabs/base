package jobs

import (
	"context"
	"log"
	"sync"
	"time"
)

// Schedule represents a recurring task schedule.
type Schedule struct {
	Name     string
	Interval time.Duration
	JobType  string // matches a registered job handler
	Payload  any    // JSON-marshaled as job payload
}

// Every creates a Schedule with the given interval.
func Every(interval time.Duration, jobType string, payload any) Schedule {
	return Schedule{
		Name:     jobType,
		Interval: interval,
		JobType:  jobType,
		Payload:  payload,
	}
}

// EveryMinutes creates a Schedule that runs every n minutes.
func EveryMinutes(n int, jobType string, payload any) Schedule {
	return Schedule{
		Name:     jobType,
		Interval: time.Duration(n) * time.Minute,
		JobType:  jobType,
		Payload:  payload,
	}
}

// EveryHours creates a Schedule that runs every n hours.
func EveryHours(n int, jobType string, payload any) Schedule {
	return Schedule{
		Name:     jobType,
		Interval: time.Duration(n) * time.Hour,
		JobType:  jobType,
		Payload:  payload,
	}
}

// Daily creates a Schedule that runs once per day (every 24 hours).
func Daily(jobType string, payload any) Schedule {
	return Schedule{
		Name:     jobType,
		Interval: 24 * time.Hour,
		JobType:  jobType,
		Payload:  payload,
	}
}

// Scheduler runs recurring tasks by enqueuing jobs at specified intervals.
type Scheduler struct {
	queue     *Queue
	schedules []Schedule
	mu        sync.Mutex
}

// NewScheduler creates a new Scheduler backed by the given Queue.
func NewScheduler(queue *Queue) *Scheduler {
	return &Scheduler{
		queue: queue,
	}
}

// Add registers a recurring task.
func (s *Scheduler) Add(schedule Schedule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.schedules = append(s.schedules, schedule)
}

// Start begins the scheduler loop. It blocks until the context is cancelled.
// For each schedule, a goroutine is launched with a time.Ticker. The first
// job is enqueued immediately (without waiting for the first tick).
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	schedules := make([]Schedule, len(s.schedules))
	copy(schedules, s.schedules)
	s.mu.Unlock()

	var wg sync.WaitGroup

	for _, sched := range schedules {
		wg.Add(1)
		go func(sc Schedule) {
			defer wg.Done()
			s.runSchedule(ctx, sc)
		}(sched)
	}

	wg.Wait()
}

// runSchedule enqueues the job immediately, then repeats on each tick until
// the context is cancelled.
func (s *Scheduler) runSchedule(ctx context.Context, sc Schedule) {
	// Enqueue immediately on start.
	if err := s.queue.Enqueue(ctx, sc.JobType, sc.Payload); err != nil {
		log.Printf("scheduler: failed to enqueue %q: %v", sc.JobType, err)
	}

	ticker := time.NewTicker(sc.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.queue.Enqueue(ctx, sc.JobType, sc.Payload); err != nil {
				log.Printf("scheduler: failed to enqueue %q: %v", sc.JobType, err)
			}
		}
	}
}
