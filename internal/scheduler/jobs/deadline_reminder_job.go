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

// DeadlineReminderJob sends reminders for upcoming assignment and quiz deadlines
type DeadlineReminderJob struct {
	assignmentRepo   *postgres.AssignmentRepository
	quizRepo         *postgres.QuizRepository
	enrollmentRepo   *postgres.EnrollmentRepository
	courseRepo       *postgres.CourseRepository
	notificationRepo *postgres.NotificationRepository
	logger           *log.Logger
	interval         time.Duration
	reminderHours    []int // Send reminders at these hours before deadline
}

// NewDeadlineReminderJob creates a new deadline reminder job
func NewDeadlineReminderJob(
	assignmentRepo *postgres.AssignmentRepository,
	quizRepo *postgres.QuizRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	courseRepo *postgres.CourseRepository,
	notificationRepo *postgres.NotificationRepository,
	logger *log.Logger,
) *DeadlineReminderJob {
	if logger == nil {
		logger = log.Default()
	}
	return &DeadlineReminderJob{
		assignmentRepo:   assignmentRepo,
		quizRepo:         quizRepo,
		enrollmentRepo:   enrollmentRepo,
		courseRepo:       courseRepo,
		notificationRepo: notificationRepo,
		logger:           logger,
		interval:         30 * time.Minute, // Check every 30 minutes
		reminderHours:    []int{24, 6, 1},  // Send reminders 24h, 6h, and 1h before
	}
}

// Name returns the job name
func (j *DeadlineReminderJob) Name() string {
	return "deadline-reminder"
}

// Interval returns how often the job should run
func (j *DeadlineReminderJob) Interval() time.Duration {
	return j.interval
}

// Run executes the job
func (j *DeadlineReminderJob) Run(ctx context.Context) error {
	j.logger.Println("[DeadlineReminderJob] Checking for upcoming deadlines")

	var totalReminders int

	// Process assignment deadlines
	assignmentReminders, err := j.processAssignmentDeadlines(ctx)
	if err != nil {
		j.logger.Printf("[DeadlineReminderJob] Error processing assignments: %v", err)
	} else {
		totalReminders += assignmentReminders
	}

	// Process quiz deadlines
	quizReminders, err := j.processQuizDeadlines(ctx)
	if err != nil {
		j.logger.Printf("[DeadlineReminderJob] Error processing quizzes: %v", err)
	} else {
		totalReminders += quizReminders
	}

	j.logger.Printf("[DeadlineReminderJob] Sent %d deadline reminders", totalReminders)
	return nil
}

// processAssignmentDeadlines checks for upcoming assignment deadlines
func (j *DeadlineReminderJob) processAssignmentDeadlines(ctx context.Context) (int, error) {
	if j.assignmentRepo == nil {
		return 0, nil
	}

	now := time.Now()
	var totalReminders int

	for _, hours := range j.reminderHours {
		select {
		case <-ctx.Done():
			return totalReminders, ctx.Err()
		default:
		}

		// Calculate the time window
		windowStart := now.Add(time.Duration(hours) * time.Hour)
		windowEnd := windowStart.Add(j.interval)

		// Get assignments with deadlines in this window
		assignments, err := j.assignmentRepo.ListByDeadlineRange(ctx, windowStart, windowEnd)
		if err != nil {
			j.logger.Printf("[DeadlineReminderJob] Failed to list assignments: %v", err)
			continue
		}

		for _, assignment := range assignments {
			if assignment.Status != domain.AssessmentStatusPublished {
				continue
			}

			count, err := j.sendAssignmentReminders(ctx, assignment, hours)
			if err != nil {
				j.logger.Printf("[DeadlineReminderJob] Failed to send reminders for assignment %s: %v", assignment.ID, err)
				continue
			}
			totalReminders += count
		}
	}

	return totalReminders, nil
}

// processQuizDeadlines checks for upcoming quiz availability end times
func (j *DeadlineReminderJob) processQuizDeadlines(ctx context.Context) (int, error) {
	if j.quizRepo == nil {
		return 0, nil
	}

	now := time.Now()
	var totalReminders int

	for _, hours := range j.reminderHours {
		select {
		case <-ctx.Done():
			return totalReminders, ctx.Err()
		default:
		}

		// Calculate the time window
		windowStart := now.Add(time.Duration(hours) * time.Hour)
		windowEnd := windowStart.Add(j.interval)

		// Get quizzes with availability ending in this window
		quizzes, err := j.quizRepo.ListByAvailabilityEndRange(ctx, windowStart, windowEnd)
		if err != nil {
			j.logger.Printf("[DeadlineReminderJob] Failed to list quizzes: %v", err)
			continue
		}

		for _, quiz := range quizzes {
			if quiz.Status != domain.AssessmentStatusPublished {
				continue
			}

			count, err := j.sendQuizReminders(ctx, quiz, hours)
			if err != nil {
				j.logger.Printf("[DeadlineReminderJob] Failed to send reminders for quiz %s: %v", quiz.ID, err)
				continue
			}
			totalReminders += count
		}
	}

	return totalReminders, nil
}

// sendAssignmentReminders sends reminders for an assignment deadline
func (j *DeadlineReminderJob) sendAssignmentReminders(ctx context.Context, assignment *domain.Assignment, hoursBefore int) (int, error) {
	if j.notificationRepo == nil || j.courseRepo == nil || j.enrollmentRepo == nil {
		return 0, nil
	}

	// Get the course to find the class
	course, err := j.courseRepo.Get(ctx, assignment.CourseID)
	if err != nil {
		return 0, fmt.Errorf("failed to get course: %w", err)
	}

	// Get enrolled students
	enrollments, _, err := j.enrollmentRepo.ListByClass(ctx, course.ClassID, repository.PaginationParams{Limit: 1000})
	if err != nil {
		return 0, fmt.Errorf("failed to get enrollments: %w", err)
	}

	// TODO: Filter out students who have already submitted

	now := time.Now()
	var title, body string
	if hoursBefore >= 24 {
		title = "Assignment Due Tomorrow"
		body = fmt.Sprintf("'%s' is due in %d hours. Submit before the deadline!", assignment.Title, hoursBefore)
	} else {
		title = fmt.Sprintf("Assignment Due in %d Hours", hoursBefore)
		body = fmt.Sprintf("'%s' deadline is approaching. Submit now!", assignment.Title)
	}

	actionURL := fmt.Sprintf("/assignments/%s", assignment.ID)
	resourceType := "ASSIGNMENT"

	var count int
	for _, enrollment := range enrollments {
		notification := &domain.Notification{
			ID:           uuid.New(),
			TenantID:     assignment.TenantID,
			UserID:       enrollment.StudentID,
			EventType:    domain.NotificationEventAssignmentDeadline,
			Title:        title,
			Body:         body,
			Channels:     []domain.NotificationChannel{domain.NotificationChannelInApp, domain.NotificationChannelPush},
			Priority:     domain.NotificationPriorityHigh,
			ActionURL:    &actionURL,
			ResourceType: &resourceType,
			ResourceID:   &assignment.ID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := j.notificationRepo.Create(ctx, notification); err != nil {
			j.logger.Printf("[DeadlineReminderJob] Failed to create notification: %v", err)
			continue
		}
		count++
	}

	return count, nil
}

// sendQuizReminders sends reminders for a quiz availability end
func (j *DeadlineReminderJob) sendQuizReminders(ctx context.Context, quiz *domain.Quiz, hoursBefore int) (int, error) {
	if j.notificationRepo == nil || j.courseRepo == nil || j.enrollmentRepo == nil {
		return 0, nil
	}

	// Get the course to find the class
	course, err := j.courseRepo.Get(ctx, quiz.CourseID)
	if err != nil {
		return 0, fmt.Errorf("failed to get course: %w", err)
	}

	// Get enrolled students
	enrollments, _, err := j.enrollmentRepo.ListByClass(ctx, course.ClassID, repository.PaginationParams{Limit: 1000})
	if err != nil {
		return 0, fmt.Errorf("failed to get enrollments: %w", err)
	}

	// TODO: Filter out students who have already completed the quiz

	now := time.Now()
	var title, body string
	if hoursBefore >= 24 {
		title = "Quiz Closing Tomorrow"
		body = fmt.Sprintf("'%s' will be unavailable in %d hours. Complete it before time runs out!", quiz.Title, hoursBefore)
	} else {
		title = fmt.Sprintf("Quiz Closing in %d Hours", hoursBefore)
		body = fmt.Sprintf("'%s' is closing soon. Take it now!", quiz.Title)
	}

	actionURL := fmt.Sprintf("/quizzes/%s", quiz.ID)
	resourceType := "QUIZ"

	var count int
	for _, enrollment := range enrollments {
		notification := &domain.Notification{
			ID:           uuid.New(),
			TenantID:     quiz.TenantID,
			UserID:       enrollment.StudentID,
			EventType:    domain.NotificationEventAssignmentDeadline, // Reusing this event type
			Title:        title,
			Body:         body,
			Channels:     []domain.NotificationChannel{domain.NotificationChannelInApp, domain.NotificationChannelPush},
			Priority:     domain.NotificationPriorityHigh,
			ActionURL:    &actionURL,
			ResourceType: &resourceType,
			ResourceID:   &quiz.ID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := j.notificationRepo.Create(ctx, notification); err != nil {
			j.logger.Printf("[DeadlineReminderJob] Failed to create notification: %v", err)
			continue
		}
		count++
	}

	return count, nil
}
