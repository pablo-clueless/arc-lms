package jobs

import (
	"context"
	"log"
	"time"

	"arc-lms/internal/repository/postgres"
)

// NotificationCleanupJob cleans up old read notifications
type NotificationCleanupJob struct {
	notificationRepo *postgres.NotificationRepository
	logger           *log.Logger
	interval         time.Duration
	retentionDays    int // Keep read notifications for this many days
}

// NewNotificationCleanupJob creates a new notification cleanup job
func NewNotificationCleanupJob(
	notificationRepo *postgres.NotificationRepository,
	logger *log.Logger,
) *NotificationCleanupJob {
	if logger == nil {
		logger = log.Default()
	}
	return &NotificationCleanupJob{
		notificationRepo: notificationRepo,
		logger:           logger,
		interval:         24 * time.Hour, // Run daily
		retentionDays:    30,             // Keep read notifications for 30 days
	}
}

// Name returns the job name
func (j *NotificationCleanupJob) Name() string {
	return "notification-cleanup"
}

// Interval returns how often the job should run
func (j *NotificationCleanupJob) Interval() time.Duration {
	return j.interval
}

// Run executes the job
func (j *NotificationCleanupJob) Run(ctx context.Context) error {
	j.logger.Println("[NotificationCleanupJob] Starting cleanup of old notifications")

	cutoffDate := time.Now().AddDate(0, 0, -j.retentionDays)

	deletedCount, err := j.notificationRepo.DeleteOldReadNotifications(ctx, cutoffDate)
	if err != nil {
		j.logger.Printf("[NotificationCleanupJob] Failed to delete old notifications: %v", err)
		return err
	}

	j.logger.Printf("[NotificationCleanupJob] Deleted %d old read notifications", deletedCount)
	return nil
}
