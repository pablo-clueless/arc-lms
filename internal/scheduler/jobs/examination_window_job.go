package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// ExaminationWindowJob manages examination window status changes and notifications
type ExaminationWindowJob struct {
	examinationRepo  *postgres.ExaminationRepository
	enrollmentRepo   *postgres.EnrollmentRepository
	courseRepo       *postgres.CourseRepository
	notificationRepo *postgres.NotificationRepository
	logger           *log.Logger
	interval         time.Duration
}

// NewExaminationWindowJob creates a new examination window job
func NewExaminationWindowJob(
	examinationRepo *postgres.ExaminationRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	courseRepo *postgres.CourseRepository,
	notificationRepo *postgres.NotificationRepository,
	logger *log.Logger,
) *ExaminationWindowJob {
	if logger == nil {
		logger = log.Default()
	}
	return &ExaminationWindowJob{
		examinationRepo:  examinationRepo,
		enrollmentRepo:   enrollmentRepo,
		courseRepo:       courseRepo,
		notificationRepo: notificationRepo,
		logger:           logger,
		interval:         1 * time.Minute, // Check every minute for exam windows
	}
}

// Name returns the job name
func (j *ExaminationWindowJob) Name() string {
	return "examination-window-manager"
}

// Interval returns how often the job should run
func (j *ExaminationWindowJob) Interval() time.Duration {
	return j.interval
}

// Run executes the job
func (j *ExaminationWindowJob) Run(ctx context.Context) error {
	j.logger.Println("[ExaminationWindowJob] Checking examination windows")

	now := time.Now()

	// Process examinations that should start (window just opened)
	startedCount, err := j.processWindowOpenings(ctx, now)
	if err != nil {
		j.logger.Printf("[ExaminationWindowJob] Error processing window openings: %v", err)
	}

	// Process examinations that should end (window just closed)
	endedCount, err := j.processWindowClosings(ctx, now)
	if err != nil {
		j.logger.Printf("[ExaminationWindowJob] Error processing window closings: %v", err)
	}

	if startedCount > 0 || endedCount > 0 {
		j.logger.Printf("[ExaminationWindowJob] Processed %d window openings, %d window closings", startedCount, endedCount)
	}

	return nil
}

// processWindowOpenings handles examinations whose window just opened
func (j *ExaminationWindowJob) processWindowOpenings(ctx context.Context, now time.Time) (int, error) {
	if j.examinationRepo == nil {
		return 0, nil
	}

	// Get scheduled examinations whose window has started
	examinations, err := j.examinationRepo.ListByWindowStart(ctx, now.Add(-j.interval), now)
	if err != nil {
		return 0, fmt.Errorf("failed to list examinations: %w", err)
	}

	var count int
	for _, exam := range examinations {
		if exam.Status != domain.ExaminationStatusScheduled {
			continue
		}

		// Update status to IN_PROGRESS
		exam.Status = domain.ExaminationStatusInProgress
		exam.UpdatedAt = now

		if err := j.examinationRepo.Update(ctx, exam); err != nil {
			j.logger.Printf("[ExaminationWindowJob] Failed to update examination %s: %v", exam.ID, err)
			continue
		}

		// Send notifications to students
		if err := j.notifyWindowOpen(ctx, exam); err != nil {
			j.logger.Printf("[ExaminationWindowJob] Failed to send window open notifications for %s: %v", exam.ID, err)
		}

		count++
	}

	return count, nil
}

// processWindowClosings handles examinations whose window just closed
func (j *ExaminationWindowJob) processWindowClosings(ctx context.Context, now time.Time) (int, error) {
	if j.examinationRepo == nil {
		return 0, nil
	}

	// Get in-progress examinations whose window has ended
	examinations, err := j.examinationRepo.ListByWindowEnd(ctx, now.Add(-j.interval), now)
	if err != nil {
		return 0, fmt.Errorf("failed to list examinations: %w", err)
	}

	var count int
	for _, exam := range examinations {
		if exam.Status != domain.ExaminationStatusInProgress {
			continue
		}

		// Update status to COMPLETED
		exam.Status = domain.ExaminationStatusCompleted
		exam.UpdatedAt = now

		if err := j.examinationRepo.Update(ctx, exam); err != nil {
			j.logger.Printf("[ExaminationWindowJob] Failed to update examination %s: %v", exam.ID, err)
			continue
		}

		// Auto-submit any in-progress submissions
		if err := j.autoSubmitPending(ctx, exam.ID); err != nil {
			j.logger.Printf("[ExaminationWindowJob] Failed to auto-submit pending for %s: %v", exam.ID, err)
		}

		// Send notifications about window closing
		if err := j.notifyWindowClose(ctx, exam); err != nil {
			j.logger.Printf("[ExaminationWindowJob] Failed to send window close notifications for %s: %v", exam.ID, err)
		}

		count++
	}

	return count, nil
}

// notifyWindowOpen sends notifications when an exam window opens
func (j *ExaminationWindowJob) notifyWindowOpen(ctx context.Context, exam *domain.Examination) error {
	if j.notificationRepo == nil || j.courseRepo == nil || j.enrollmentRepo == nil {
		return nil
	}

	// Get the course to find the class
	course, err := j.courseRepo.Get(ctx, exam.CourseID)
	if err != nil {
		return fmt.Errorf("failed to get course: %w", err)
	}

	// Get enrolled students
	enrollments, _, err := j.enrollmentRepo.ListByClass(ctx, course.ClassID, repository.PaginationParams{Limit: 1000})
	if err != nil {
		return fmt.Errorf("failed to get enrollments: %w", err)
	}

	now := time.Now()
	title := "Examination Now Open"
	body := fmt.Sprintf("The examination '%s' is now available. Window closes at %s.",
		exam.Title,
		exam.WindowEnd.Format("Jan 2, 2006 3:04 PM"))
	actionURL := fmt.Sprintf("/examinations/%s", exam.ID)
	resourceType := "EXAMINATION"

	for _, enrollment := range enrollments {
		notification := &domain.Notification{
			ID:           uuid.New(),
			TenantID:     exam.TenantID,
			UserID:       enrollment.StudentID,
			EventType:    domain.NotificationEventExaminationWindowOpen,
			Title:        title,
			Body:         body,
			Channels:     []domain.NotificationChannel{domain.NotificationChannelInApp, domain.NotificationChannelPush, domain.NotificationChannelEmail},
			Priority:     domain.NotificationPriorityUrgent,
			ActionURL:    &actionURL,
			ResourceType: &resourceType,
			ResourceID:   &exam.ID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := j.notificationRepo.Create(ctx, notification); err != nil {
			j.logger.Printf("[ExaminationWindowJob] Failed to create notification: %v", err)
		}
	}

	return nil
}

// notifyWindowClose sends notifications when an exam window closes
func (j *ExaminationWindowJob) notifyWindowClose(ctx context.Context, exam *domain.Examination) error {
	if j.notificationRepo == nil || j.courseRepo == nil || j.enrollmentRepo == nil {
		return nil
	}

	// Get the course to find the class
	course, err := j.courseRepo.Get(ctx, exam.CourseID)
	if err != nil {
		return fmt.Errorf("failed to get course: %w", err)
	}

	// Get enrolled students
	enrollments, _, err := j.enrollmentRepo.ListByClass(ctx, course.ClassID, repository.PaginationParams{Limit: 1000})
	if err != nil {
		return fmt.Errorf("failed to get enrollments: %w", err)
	}

	now := time.Now()
	title := "Examination Window Closed"
	body := fmt.Sprintf("The examination '%s' window has closed. Results will be published soon.", exam.Title)
	resourceType := "EXAMINATION"

	for _, enrollment := range enrollments {
		notification := &domain.Notification{
			ID:           uuid.New(),
			TenantID:     exam.TenantID,
			UserID:       enrollment.StudentID,
			EventType:    domain.NotificationEventExaminationWindowClose,
			Title:        title,
			Body:         body,
			Channels:     []domain.NotificationChannel{domain.NotificationChannelInApp},
			Priority:     domain.NotificationPriorityNormal,
			ResourceType: &resourceType,
			ResourceID:   &exam.ID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := j.notificationRepo.Create(ctx, notification); err != nil {
			j.logger.Printf("[ExaminationWindowJob] Failed to create notification: %v", err)
		}
	}

	return nil
}

// autoSubmitPending auto-submits any in-progress examination submissions
func (j *ExaminationWindowJob) autoSubmitPending(ctx context.Context, examID uuid.UUID) error {
	if j.examinationRepo == nil {
		return nil
	}

	count, err := j.examinationRepo.AutoSubmitInProgress(ctx, examID)
	if err != nil {
		return err
	}

	if count > 0 {
		j.logger.Printf("[ExaminationWindowJob] Auto-submitted %d in-progress submissions for exam %s", count, examID)
	}

	return nil
}
