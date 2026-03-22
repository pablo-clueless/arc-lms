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

// MeetingReminderJob sends reminders for upcoming meetings
type MeetingReminderJob struct {
	meetingRepo      *postgres.MeetingRepository
	enrollmentRepo   *postgres.EnrollmentRepository
	notificationRepo *postgres.NotificationRepository
	logger           *log.Logger
	interval         time.Duration
	reminderMinutes  []int // Send reminders at these intervals before meeting
}

// NewMeetingReminderJob creates a new meeting reminder job
func NewMeetingReminderJob(
	meetingRepo *postgres.MeetingRepository,
	enrollmentRepo *postgres.EnrollmentRepository,
	notificationRepo *postgres.NotificationRepository,
	logger *log.Logger,
) *MeetingReminderJob {
	if logger == nil {
		logger = log.Default()
	}
	return &MeetingReminderJob{
		meetingRepo:      meetingRepo,
		enrollmentRepo:   enrollmentRepo,
		notificationRepo: notificationRepo,
		logger:           logger,
		interval:         5 * time.Minute, // Check every 5 minutes
		reminderMinutes:  []int{15, 5},    // Send reminders 15 and 5 minutes before
	}
}

// Name returns the job name
func (j *MeetingReminderJob) Name() string {
	return "meeting-reminder"
}

// Interval returns how often the job should run
func (j *MeetingReminderJob) Interval() time.Duration {
	return j.interval
}

// Run executes the job
func (j *MeetingReminderJob) Run(ctx context.Context) error {
	j.logger.Println("[MeetingReminderJob] Checking for upcoming meetings")

	now := time.Now()
	var totalReminders int

	for _, minutes := range j.reminderMinutes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate the time window for this reminder
		// e.g., for 15-minute reminder: look for meetings between 15-16 minutes from now
		windowStart := now.Add(time.Duration(minutes) * time.Minute)
		windowEnd := windowStart.Add(time.Duration(j.interval))

		// Get meetings in this window
		meetings, _, err := j.meetingRepo.ListByDateRange(ctx, uuid.Nil, windowStart, windowEnd, repository.PaginationParams{Limit: 100})
		if err != nil {
			j.logger.Printf("[MeetingReminderJob] Failed to list meetings: %v", err)
			continue
		}

		for _, meeting := range meetings {
			if meeting.Status != domain.MeetingStatusScheduled {
				continue
			}

			count, err := j.sendMeetingReminders(ctx, meeting, minutes)
			if err != nil {
				j.logger.Printf("[MeetingReminderJob] Failed to send reminders for meeting %s: %v", meeting.ID, err)
				continue
			}
			totalReminders += count
		}
	}

	j.logger.Printf("[MeetingReminderJob] Sent %d meeting reminders", totalReminders)
	return nil
}

// sendMeetingReminders sends reminders for a specific meeting
func (j *MeetingReminderJob) sendMeetingReminders(ctx context.Context, meeting *domain.Meeting, minutesBefore int) (int, error) {
	if j.notificationRepo == nil || j.enrollmentRepo == nil {
		return 0, nil
	}

	// Get enrolled students in the class
	enrollments, _, err := j.enrollmentRepo.ListByClass(ctx, meeting.ClassID, repository.PaginationParams{Limit: 1000})
	if err != nil {
		return 0, fmt.Errorf("failed to get enrollments: %w", err)
	}

	now := time.Now()
	title := fmt.Sprintf("Meeting Starting in %d Minutes", minutesBefore)
	body := fmt.Sprintf("'%s' is starting soon. Click to join.", meeting.Title)
	actionURL := fmt.Sprintf("/meetings/%s/join", meeting.ID)
	resourceType := "MEETING"

	var count int
	for _, enrollment := range enrollments {
		notification := &domain.Notification{
			ID:           uuid.New(),
			TenantID:     meeting.TenantID,
			UserID:       enrollment.StudentID,
			EventType:    domain.NotificationEventMeetingStarting,
			Title:        title,
			Body:         body,
			Channels:     []domain.NotificationChannel{domain.NotificationChannelInApp, domain.NotificationChannelPush},
			Priority:     domain.NotificationPriorityHigh,
			ActionURL:    &actionURL,
			ResourceType: &resourceType,
			ResourceID:   &meeting.ID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := j.notificationRepo.Create(ctx, notification); err != nil {
			j.logger.Printf("[MeetingReminderJob] Failed to create notification for user %s: %v", enrollment.StudentID, err)
			continue
		}
		count++
	}

	// Also notify the host tutor
	tutorNotification := &domain.Notification{
		ID:           uuid.New(),
		TenantID:     meeting.TenantID,
		UserID:       meeting.HostTutorID,
		EventType:    domain.NotificationEventMeetingStarting,
		Title:        title,
		Body:         fmt.Sprintf("Your meeting '%s' is starting in %d minutes.", meeting.Title, minutesBefore),
		Channels:     []domain.NotificationChannel{domain.NotificationChannelInApp, domain.NotificationChannelPush},
		Priority:     domain.NotificationPriorityHigh,
		ActionURL:    &actionURL,
		ResourceType: &resourceType,
		ResourceID:   &meeting.ID,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := j.notificationRepo.Create(ctx, tutorNotification); err != nil {
		j.logger.Printf("[MeetingReminderJob] Failed to create notification for tutor %s: %v", meeting.HostTutorID, err)
	} else {
		count++
	}

	return count, nil
}
