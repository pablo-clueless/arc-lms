package scheduler

import (
	"database/sql"
	"log"

	"arc-lms/internal/pkg/email"
	"arc-lms/internal/repository/postgres"
	"arc-lms/internal/scheduler/jobs"
)

// SchedulerConfig holds configuration for the scheduler
type SchedulerConfig struct {
	DB           *sql.DB
	EmailService *email.EmailService
	Logger       *log.Logger
}

// SetupScheduler creates and configures the scheduler with all background jobs
func SetupScheduler(cfg *SchedulerConfig) *Scheduler {
	scheduler := NewScheduler(cfg.Logger)

	// Initialize repositories needed by jobs
	invoiceRepo := postgres.NewInvoiceRepository(cfg.DB)
	subscriptionRepo := postgres.NewSubscriptionRepository(cfg.DB)
	tenantRepo := postgres.NewTenantRepository(cfg.DB)
	communicationRepo := postgres.NewCommunicationRepository(cfg.DB)
	notificationRepo := postgres.NewNotificationRepository(cfg.DB)
	meetingRepo := postgres.NewMeetingRepository(cfg.DB)
	enrollmentRepo := postgres.NewEnrollmentRepository(cfg.DB)
	assignmentRepo := postgres.NewAssignmentRepository(cfg.DB)
	quizRepo := postgres.NewQuizRepository(cfg.DB)
	courseRepo := postgres.NewCourseRepository(cfg.DB)
	examinationRepo := postgres.NewExaminationRepository(cfg.DB)

	// Register billing/invoice jobs
	overdueInvoiceJob := jobs.NewOverdueInvoiceJob(invoiceRepo, subscriptionRepo, tenantRepo, notificationRepo, cfg.Logger)
	scheduler.Register(overdueInvoiceJob)

	// Register scheduled email job
	scheduledEmailJob := jobs.NewScheduledEmailJob(communicationRepo, cfg.EmailService, cfg.Logger)
	scheduler.Register(scheduledEmailJob)

	// Register meeting reminder job
	meetingReminderJob := jobs.NewMeetingReminderJob(meetingRepo, enrollmentRepo, notificationRepo, cfg.Logger)
	scheduler.Register(meetingReminderJob)

	// Register deadline reminder job
	deadlineReminderJob := jobs.NewDeadlineReminderJob(
		assignmentRepo,
		quizRepo,
		enrollmentRepo,
		courseRepo,
		notificationRepo,
		cfg.Logger,
	)
	scheduler.Register(deadlineReminderJob)

	// Register notification cleanup job
	notificationCleanupJob := jobs.NewNotificationCleanupJob(notificationRepo, cfg.Logger)
	scheduler.Register(notificationCleanupJob)

	// Register examination window job
	examinationWindowJob := jobs.NewExaminationWindowJob(
		examinationRepo,
		enrollmentRepo,
		courseRepo,
		notificationRepo,
		cfg.Logger,
	)
	scheduler.Register(examinationWindowJob)

	return scheduler
}
