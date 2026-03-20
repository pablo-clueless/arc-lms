package scheduler

import (
	"database/sql"
	"log"

	"arc-lms/internal/repository/postgres"
	"arc-lms/internal/scheduler/jobs"
)

// SetupScheduler creates and configures the scheduler with all background jobs
func SetupScheduler(db *sql.DB, logger *log.Logger) *Scheduler {
	scheduler := NewScheduler(logger)

	// Initialize repositories needed by jobs
	invoiceRepo := postgres.NewInvoiceRepository(db)
	subscriptionRepo := postgres.NewSubscriptionRepository(db)
	tenantRepo := postgres.NewTenantRepository(db)
	communicationRepo := postgres.NewCommunicationRepository(db)
	notificationRepo := postgres.NewNotificationRepository(db)
	meetingRepo := postgres.NewMeetingRepository(db)
	enrollmentRepo := postgres.NewEnrollmentRepository(db)
	assignmentRepo := postgres.NewAssignmentRepository(db)
	quizRepo := postgres.NewQuizRepository(db)
	courseRepo := postgres.NewCourseRepository(db)
	examinationRepo := postgres.NewExaminationRepository(db)

	// Register billing/invoice jobs
	overdueInvoiceJob := jobs.NewOverdueInvoiceJob(invoiceRepo, subscriptionRepo, tenantRepo, notificationRepo, logger)
	scheduler.Register(overdueInvoiceJob)

	// Register scheduled email job
	scheduledEmailJob := jobs.NewScheduledEmailJob(communicationRepo, logger)
	scheduler.Register(scheduledEmailJob)

	// Register meeting reminder job
	meetingReminderJob := jobs.NewMeetingReminderJob(meetingRepo, enrollmentRepo, notificationRepo, logger)
	scheduler.Register(meetingReminderJob)

	// Register deadline reminder job
	deadlineReminderJob := jobs.NewDeadlineReminderJob(
		assignmentRepo,
		quizRepo,
		enrollmentRepo,
		courseRepo,
		notificationRepo,
		logger,
	)
	scheduler.Register(deadlineReminderJob)

	// Register notification cleanup job
	notificationCleanupJob := jobs.NewNotificationCleanupJob(notificationRepo, logger)
	scheduler.Register(notificationCleanupJob)

	// Register examination window job
	examinationWindowJob := jobs.NewExaminationWindowJob(
		examinationRepo,
		enrollmentRepo,
		courseRepo,
		notificationRepo,
		logger,
	)
	scheduler.Register(examinationWindowJob)

	return scheduler
}
