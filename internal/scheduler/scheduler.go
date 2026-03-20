package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Job represents a background job that can be scheduled
type Job interface {
	Name() string
	Run(ctx context.Context) error
	Interval() time.Duration
}

// Scheduler manages and runs background jobs
type Scheduler struct {
	jobs       []Job
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
	running    bool
	logger     *log.Logger
	jobStatuses map[string]*JobStatus
}

// JobStatus tracks the status of a job
type JobStatus struct {
	Name         string        `json:"name"`
	LastRun      *time.Time    `json:"last_run,omitempty"`
	LastDuration time.Duration `json:"last_duration_ms"`
	LastError    string        `json:"last_error,omitempty"`
	RunCount     int64         `json:"run_count"`
	ErrorCount   int64         `json:"error_count"`
	IsRunning    bool          `json:"is_running"`
}

// NewScheduler creates a new scheduler
func NewScheduler(logger *log.Logger) *Scheduler {
	if logger == nil {
		logger = log.Default()
	}
	return &Scheduler{
		jobs:        make([]Job, 0),
		logger:      logger,
		jobStatuses: make(map[string]*JobStatus),
	}
}

// Register adds a job to the scheduler
func (s *Scheduler) Register(job Job) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.jobs = append(s.jobs, job)
	s.jobStatuses[job.Name()] = &JobStatus{
		Name: job.Name(),
	}
	s.logger.Printf("[Scheduler] Registered job: %s (interval: %s)", job.Name(), job.Interval())
}

// Start begins running all registered jobs
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.mu.Unlock()

	s.logger.Printf("[Scheduler] Starting with %d jobs", len(s.jobs))

	for _, job := range s.jobs {
		s.wg.Add(1)
		go s.runJob(job)
	}
}

// Stop gracefully stops all running jobs
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	s.logger.Println("[Scheduler] Stopping all jobs...")
	s.cancel()
	s.wg.Wait()
	s.logger.Println("[Scheduler] All jobs stopped")
}

// runJob runs a single job on its interval
func (s *Scheduler) runJob(job Job) {
	defer s.wg.Done()

	ticker := time.NewTicker(job.Interval())
	defer ticker.Stop()

	// Run immediately on start
	s.executeJob(job)

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Printf("[Scheduler] Job %s shutting down", job.Name())
			return
		case <-ticker.C:
			s.executeJob(job)
		}
	}
}

// executeJob executes a single job and tracks its status
func (s *Scheduler) executeJob(job Job) {
	s.mu.Lock()
	status := s.jobStatuses[job.Name()]
	if status.IsRunning {
		s.mu.Unlock()
		s.logger.Printf("[Scheduler] Job %s is already running, skipping", job.Name())
		return
	}
	status.IsRunning = true
	s.mu.Unlock()

	startTime := time.Now()
	s.logger.Printf("[Scheduler] Starting job: %s", job.Name())

	err := job.Run(s.ctx)

	duration := time.Since(startTime)

	s.mu.Lock()
	status.IsRunning = false
	status.LastRun = &startTime
	status.LastDuration = duration
	status.RunCount++
	if err != nil {
		status.ErrorCount++
		status.LastError = err.Error()
		s.logger.Printf("[Scheduler] Job %s failed after %s: %v", job.Name(), duration, err)
	} else {
		status.LastError = ""
		s.logger.Printf("[Scheduler] Job %s completed in %s", job.Name(), duration)
	}
	s.mu.Unlock()
}

// GetStatus returns the status of all jobs
func (s *Scheduler) GetStatus() map[string]*JobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*JobStatus)
	for name, status := range s.jobStatuses {
		copyStatus := *status
		result[name] = &copyStatus
	}
	return result
}

// GetJobStatus returns the status of a specific job
func (s *Scheduler) GetJobStatus(name string) (*JobStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status, exists := s.jobStatuses[name]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", name)
	}
	copyStatus := *status
	return &copyStatus, nil
}

// RunJobNow triggers a job to run immediately (for manual triggers)
func (s *Scheduler) RunJobNow(name string) error {
	s.mu.RLock()
	var targetJob Job
	for _, job := range s.jobs {
		if job.Name() == name {
			targetJob = job
			break
		}
	}
	s.mu.RUnlock()

	if targetJob == nil {
		return fmt.Errorf("job not found: %s", name)
	}

	go s.executeJob(targetJob)
	return nil
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}
