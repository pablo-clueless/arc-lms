package router

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"arc-lms/internal/handler"
	"arc-lms/internal/middleware"
	"arc-lms/internal/pkg/idempotency"
	"arc-lms/internal/pkg/jwt"
	customLogger "arc-lms/internal/pkg/logger"
	"arc-lms/internal/repository/postgres"
	"arc-lms/internal/service"
)

type RouterConfig struct {
	DB             *sql.DB
	RedisClient    *redis.Client
	JWTManager     *jwt.Manager
	Environment    string
	AllowedOrigins []string
}

func SetupRouter(cfg *RouterConfig) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	var logger *logrus.Logger
	logger = customLogger.NewColoredLogger()

	router.Use(middleware.RecoveryMiddleware(logger))
	router.Use(middleware.LoggerMiddleware(logger))
	router.Use(middleware.CORSMiddleware(&middleware.CORSConfig{
		AllowedOrigins: cfg.AllowedOrigins,
	}))

	router.GET("/health", healthCheckHandler)
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/api/v1/openapi.json"),
	))
	router.StaticFile("/api/v1/openapi.json", "./docs/openapi/openapi.json")

	router.GET("/redoc", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html>
<head>
	<title>Arc LMS API Documentation</title>
	<meta charset="utf-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link href="https://fonts.googleapis.com/css?family=Figtree:300,400,700" rel="stylesheet">
	<style>
		body { margin: 0; padding: 0; }
	</style>
</head>
<body>
	<redoc spec-url='/api/v1/openapi.json'></redoc>
	<script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"> </script>
</body>
</html>`
		c.String(http.StatusOK, html)
	})

	tenantRepo := postgres.NewTenantRepository(cfg.DB)
	userRepo := postgres.NewUserRepository(cfg.DB)
	sessionRepo := postgres.NewSessionRepository(cfg.DB)
	termRepo := postgres.NewTermRepository(cfg.DB)
	classRepo := postgres.NewClassRepository(cfg.DB)
	courseRepo := postgres.NewCourseRepository(cfg.DB)
	enrollmentRepo := postgres.NewEnrollmentRepository(cfg.DB)
	timetableRepo := postgres.NewTimetableRepository(cfg.DB)
	periodRepo := postgres.NewPeriodRepository(cfg.DB)
	swapRequestRepo := postgres.NewSwapRequestRepository(cfg.DB)
	auditRepo := postgres.NewAuditRepository(cfg.DB)
	notificationRepo := postgres.NewNotificationRepository(cfg.DB)
	quizRepo := postgres.NewQuizRepository(cfg.DB)
	assignmentRepo := postgres.NewAssignmentRepository(cfg.DB)
	invoiceRepo := postgres.NewInvoiceRepository(cfg.DB)
	examinationRepo := postgres.NewExaminationRepository(cfg.DB)
	progressRepo := postgres.NewProgressRepository(cfg.DB)
	meetingRepo := postgres.NewMeetingRepository(cfg.DB)
	communicationRepo := postgres.NewCommunicationRepository(cfg.DB)
	subscriptionRepo := postgres.NewSubscriptionRepository(cfg.DB)

	auditService := service.NewAuditService(auditRepo)
	billingService := service.NewBillingService(invoiceRepo, subscriptionRepo, tenantRepo, enrollmentRepo, auditService)
	authService := service.NewAuthService(userRepo, cfg.JWTManager, auditService)
	userService := service.NewUserService(userRepo, auditService)
	tenantService := service.NewTenantService(tenantRepo, userRepo, auditService)
	sessionService := service.NewSessionService(sessionRepo, auditService)
	termService := service.NewTermService(termRepo, sessionRepo, enrollmentRepo, auditService, billingService)
	classService := service.NewClassService(classRepo, sessionRepo, auditService)
	courseService := service.NewCourseService(courseRepo, classRepo, userRepo, auditService)
	enrollmentService := service.NewEnrollmentService(enrollmentRepo, classRepo, userRepo, sessionRepo, auditService)
	dashboardService := service.NewDashboardService(tenantRepo, userRepo, sessionRepo, classRepo, courseRepo, enrollmentRepo, invoiceRepo)
	notificationService := service.NewNotificationService(notificationRepo, userRepo)
	assessmentService := service.NewAssessmentService(quizRepo, assignmentRepo, courseRepo, auditService)
	timetableService := service.NewTimetableService(
		cfg.DB,
		timetableRepo,
		periodRepo,
		swapRequestRepo,
		courseRepo,
		classRepo,
		termRepo,
		tenantRepo,
		auditService,
	)
	examinationService := service.NewExaminationService(examinationRepo, courseRepo, auditService)
	progressService := service.NewProgressService(
		progressRepo,
		enrollmentRepo,
		courseRepo,
		classRepo,
		termRepo,
		quizRepo,
		assignmentRepo,
		examinationRepo,
		auditService,
	)
	meetingService := service.NewMeetingService(
		meetingRepo,
		classRepo,
		courseRepo,
		enrollmentRepo,
		notificationService,
		auditService,
	)
	communicationService := service.NewCommunicationService(
		communicationRepo,
		userRepo,
		classRepo,
		courseRepo,
		auditService,
	)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	tenantHandler := handler.NewTenantHandler(tenantService)
	sessionHandler := handler.NewSessionHandler(sessionService)
	termHandler := handler.NewTermHandler(termService)
	classHandler := handler.NewClassHandler(classService)
	courseHandler := handler.NewCourseHandler(courseService)
	enrollmentHandler := handler.NewEnrollmentHandler(enrollmentService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	auditHandler := handler.NewAuditHandler(auditService)
	notificationHandler := handler.NewNotificationHandler(notificationService)
	assessmentHandler := handler.NewAssessmentHandler(assessmentService)
	timetableHandler := handler.NewTimetableHandler(timetableService)
	examinationHandler := handler.NewExaminationHandler(examinationService)
	progressHandler := handler.NewProgressHandler(progressService)
	meetingHandler := handler.NewMeetingHandler(meetingService)
	communicationHandler := handler.NewCommunicationHandler(communicationService)
	billingHandler := handler.NewBillingHandler(billingService)

	v1 := router.Group("/api/v1")
	{
		public := v1.Group("/public")
		{
			public.POST("/auth/login", authHandler.Login)
			public.POST("/auth/register", authHandler.Register)
			public.POST("/auth/password-reset", authHandler.RequestPasswordReset)
			public.POST("/auth/password-reset/confirm", authHandler.ResetPassword)
			public.POST("/auth/refresh", authHandler.RefreshToken)
			public.GET("/auth/invitation/validate", authHandler.ValidateInvitation)
			public.POST("/auth/invitation/accept", authHandler.AcceptInvitation)
		}

		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.JWTManager))

		idempotencyStore := idempotency.NewStore(cfg.RedisClient)

		protected.Use(middleware.RateLimitMiddleware(cfg.RedisClient, &middleware.RateLimitConfig{
			TenantRateLimit:     1000,
			SuperAdminRateLimit: 2000,
			BurstMultiplier:     2,
			BurstWindow:         30,
			Window:              60,
		}))
		protected.Use(middleware.IdempotencyMiddleware(idempotencyStore))

		{

			protected.GET("/dashboard", dashboardHandler.GetDashboard)

			tenants := protected.Group("/tenants")
			{
				tenants.GET("", tenantHandler.ListTenants)
				tenants.POST("", tenantHandler.CreateTenant)
				tenants.GET("/:id", tenantHandler.GetTenant)
				tenants.PUT("/:id", tenantHandler.UpdateTenant)
				tenants.DELETE("/:id", tenantHandler.DeleteTenant)
				tenants.POST("/:id/suspend", tenantHandler.SuspendTenant)
				tenants.POST("/:id/reactivate", tenantHandler.ReactivateTenant)
				tenants.GET("/:id/configuration", tenantHandler.GetTenantConfiguration)
				tenants.PUT("/:id/configuration", tenantHandler.UpdateTenantConfiguration)
			}

			users := protected.Group("/users")
			{
				users.GET("/me", userHandler.GetMe)
				users.PUT("/me", userHandler.UpdateMe)
				users.PUT("/me/password", userHandler.ChangePassword)
				users.POST("/invite", userHandler.InviteUser)
				users.GET("", userHandler.ListUsers)
				users.GET("/:id", userHandler.GetUser)
				users.PUT("/:id", userHandler.UpdateUser)
				users.POST("/:id/deactivate", userHandler.DeactivateUser)
				users.POST("/:id/reactivate", userHandler.ReactivateUser)
			}

			// SuperAdmin management routes (SUPER_ADMIN only)
			superadmins := protected.Group("/superadmins")
			{
				superadmins.GET("", userHandler.ListSuperAdmins)
				superadmins.POST("", userHandler.CreateSuperAdmin)
				superadmins.GET("/:id", userHandler.GetSuperAdmin)
				superadmins.PUT("/:id", userHandler.UpdateSuperAdmin)
				superadmins.DELETE("/:id", userHandler.DeleteSuperAdmin)
			}

			sessions := protected.Group("/sessions")
			{
				sessions.GET("/active", sessionHandler.GetActiveSession)
				sessions.GET("", sessionHandler.ListSessions)
				sessions.POST("", sessionHandler.CreateSession)
				sessions.GET("/:id", sessionHandler.GetSession)
				sessions.PUT("/:id", sessionHandler.UpdateSession)
				sessions.DELETE("/:id", sessionHandler.DeleteSession)
				sessions.POST("/:id/activate", sessionHandler.ActivateSession)
				sessions.POST("/:id/archive", sessionHandler.ArchiveSession)
			}

			terms := protected.Group("/sessions/:id/terms")
			{
				terms.GET("/active", termHandler.GetActiveTerm)
				terms.GET("", termHandler.ListTerms)
				terms.POST("", termHandler.CreateTerm)
				terms.GET("/:term_id", termHandler.GetTerm)
				terms.PUT("/:term_id", termHandler.UpdateTerm)
				terms.DELETE("/:term_id", termHandler.DeleteTerm)
				terms.POST("/:term_id/activate", termHandler.ActivateTerm)
				terms.POST("/:term_id/complete", termHandler.CompleteTerm)
			}

			classes := protected.Group("/classes")
			{
				classes.GET("", classHandler.ListClasses)
				classes.POST("", classHandler.CreateClass)
				classes.GET("/:id", classHandler.GetClass)
				classes.PUT("/:id", classHandler.UpdateClass)
				classes.DELETE("/:id", classHandler.DeleteClass)
			}

			courses := protected.Group("/courses")
			{
				courses.GET("", courseHandler.ListCourses)
				courses.POST("", courseHandler.CreateCourse)
				courses.GET("/:id", courseHandler.GetCourse)
				courses.PUT("/:id", courseHandler.UpdateCourse)
				courses.DELETE("/:id", courseHandler.DeleteCourse)
				courses.POST("/:id/reassign-tutor", courseHandler.ReassignTutor)

				quizzes := courses.Group("/:id/quizzes")
				{
					quizzes.GET("", assessmentHandler.ListQuizzes)
					quizzes.POST("", assessmentHandler.CreateQuiz)
					quizzes.GET("/:id", assessmentHandler.GetQuiz)
					quizzes.PUT("/:id", assessmentHandler.UpdateQuiz)
					quizzes.DELETE("/:id", assessmentHandler.DeleteQuiz)
					quizzes.POST("/:id/publish", assessmentHandler.PublishQuiz)
					quizzes.POST("/:id/start", assessmentHandler.StartQuiz)
					quizzes.POST("/:id/submit", assessmentHandler.SubmitQuiz)
					quizzes.GET("/:id/submissions", assessmentHandler.ListQuizSubmissions)
					quizzes.POST("/:id/submissions/:submission_id/grade", assessmentHandler.GradeQuiz)
				}

				assignments := courses.Group("/:id/assignments")
				{
					assignments.GET("", assessmentHandler.ListAssignments)
					assignments.POST("", assessmentHandler.CreateAssignment)
					assignments.GET("/:id", assessmentHandler.GetAssignment)
					assignments.PUT("/:id", assessmentHandler.UpdateAssignment)
					assignments.DELETE("/:id", assessmentHandler.DeleteAssignment)
					assignments.POST("/:id/publish", assessmentHandler.PublishAssignment)
					assignments.POST("/:id/submit", assessmentHandler.SubmitAssignment)
					assignments.GET("/:id/submissions", assessmentHandler.ListAssignmentSubmissions)
					assignments.POST("/:id/submissions/:submission_id/grade", assessmentHandler.GradeAssignment)
				}
			}

			enrollments := protected.Group("/enrollments")
			{
				enrollments.GET("", enrollmentHandler.ListEnrollments)
				enrollments.POST("", enrollmentHandler.EnrollStudent)
				enrollments.GET("/:id", enrollmentHandler.GetEnrollment)
				enrollments.POST("/:id/transfer", enrollmentHandler.TransferStudent)
				enrollments.POST("/:id/withdraw", enrollmentHandler.WithdrawStudent)
				enrollments.POST("/:id/suspend", enrollmentHandler.SuspendEnrollment)
				enrollments.POST("/:id/reactivate", enrollmentHandler.ReactivateEnrollment)
			}

			timetables := protected.Group("/timetables")
			{
				timetables.GET("", timetableHandler.ListTimetables)
				timetables.POST("/generate", timetableHandler.GenerateTimetable)
				timetables.POST("/regenerate", timetableHandler.RegenerateTimetable)
				timetables.GET("/tutor/:tutor_id", timetableHandler.GetTutorTimetable)
				timetables.GET("/class/:class_id", timetableHandler.GetClassTimetable)
				timetables.GET("/:id", timetableHandler.GetTimetable)
				timetables.POST("/:id/publish", timetableHandler.PublishTimetable)

				// Swap request routes
				swapRequests := timetables.Group("/swap-requests")
				{
					swapRequests.GET("", timetableHandler.ListSwapRequests)
					swapRequests.POST("", timetableHandler.CreateSwapRequest)
					swapRequests.GET("/pending", timetableHandler.ListPendingSwapRequests)
					swapRequests.GET("/escalated", timetableHandler.ListEscalatedSwapRequests)
					swapRequests.GET("/:id", timetableHandler.GetSwapRequest)
					swapRequests.POST("/:id/approve", timetableHandler.ApproveSwapRequest)
					swapRequests.POST("/:id/reject", timetableHandler.RejectSwapRequest)
					swapRequests.POST("/:id/escalate", timetableHandler.EscalateSwapRequest)
					swapRequests.POST("/:id/override", timetableHandler.AdminOverrideSwapRequest)
					swapRequests.POST("/:id/cancel", timetableHandler.CancelSwapRequest)
				}
			}

			examinations := protected.Group("/examinations")
			{
				examinations.GET("", examinationHandler.ListExaminations)
				examinations.POST("", examinationHandler.CreateExamination)
				examinations.GET("/:id", examinationHandler.GetExamination)
				examinations.PUT("/:id", examinationHandler.UpdateExamination)
				examinations.DELETE("/:id", examinationHandler.DeleteExamination)
				examinations.POST("/:id/schedule", examinationHandler.ScheduleExamination)
				examinations.POST("/:id/start", examinationHandler.StartExamination)
				examinations.GET("/:id/my-submission", examinationHandler.GetMySubmission)
				examinations.GET("/:id/stats", examinationHandler.GetExaminationStats)
				examinations.POST("/:id/publish-results", examinationHandler.PublishResults)

				// Submission routes
				examinations.GET("/:id/submissions", examinationHandler.ListSubmissions)
				examinations.GET("/:id/submissions/pending-grading", examinationHandler.GetPendingGradingSubmissions)
				examinations.GET("/:id/submissions/:submission_id", examinationHandler.GetSubmission)
				examinations.POST("/:id/submissions/:submission_id/answers", examinationHandler.SaveAnswer)
				examinations.POST("/:id/submissions/:submission_id/integrity-events", examinationHandler.RecordIntegrityEvent)
				examinations.POST("/:id/submissions/:submission_id/submit", examinationHandler.SubmitExamination)
				examinations.POST("/:id/submissions/:submission_id/grade", examinationHandler.GradeSubmission)
			}

			progress := protected.Group("/progress")
			{
				// Progress records
				progress.GET("/:id", progressHandler.GetProgress)
				progress.GET("/students/:student_id", progressHandler.GetStudentProgress)
				progress.GET("/courses/:course_id", progressHandler.GetCourseProgress)
				progress.GET("/classes/:class_id", progressHandler.GetClassProgress)
				progress.GET("/flagged", progressHandler.ListFlaggedStudents)

				// Grade computation
				progress.POST("/courses/:course_id/compute-grades", progressHandler.ComputeGrades)
				progress.POST("/classes/:class_id/compute-positions", progressHandler.ComputeClassPositions)

				// Attendance
				progress.POST("/courses/:course_id/attendance", progressHandler.MarkAttendance)

				// Remarks
				progress.POST("/:id/tutor-remarks", progressHandler.AddTutorRemarks)
				progress.POST("/:id/principal-remarks", progressHandler.AddPrincipalRemarks)
				progress.POST("/:id/unflag", progressHandler.UnflagProgress)

				// Statistics
				progress.GET("/courses/:course_id/statistics", progressHandler.GetCourseStatistics)
				progress.GET("/classes/:class_id/statistics", progressHandler.GetClassStatistics)

				// Report Cards
				reportCards := progress.Group("/report-cards")
				{
					reportCards.GET("/:id", progressHandler.GetReportCard)
					reportCards.PUT("/:id/remarks", progressHandler.UpdateReportCardRemarks)
					reportCards.GET("/students/:student_id", progressHandler.GetStudentReportCard)
					reportCards.POST("/students/:student_id", progressHandler.GenerateReportCard)
					reportCards.POST("/classes/:class_id", progressHandler.GenerateClassReportCards)
				}
			}

			meetings := protected.Group("/meetings")
			{
				meetings.GET("", meetingHandler.ListMeetings)
				meetings.POST("", meetingHandler.ScheduleMeeting)
				meetings.GET("/upcoming", meetingHandler.ListUpcomingMeetings)
				meetings.GET("/live", meetingHandler.ListLiveMeetings)
				meetings.GET("/statistics", meetingHandler.GetMeetingStatistics)
				meetings.GET("/class/:class_id", meetingHandler.ListMeetingsByClass)
				meetings.GET("/:id", meetingHandler.GetMeeting)
				meetings.PUT("/:id", meetingHandler.UpdateMeeting)
				meetings.POST("/:id/start", meetingHandler.StartMeeting)
				meetings.POST("/:id/end", meetingHandler.EndMeeting)
				meetings.POST("/:id/cancel", meetingHandler.CancelMeeting)
				meetings.GET("/:id/join", meetingHandler.JoinMeeting)
				meetings.POST("/:id/participants/join", meetingHandler.RecordParticipantJoin)
				meetings.POST("/:id/participants/leave", meetingHandler.RecordParticipantLeave)
				meetings.POST("/:id/recording", meetingHandler.AddRecording)
			}

			communications := protected.Group("/communications")
			{
				// Email routes
				communications.GET("/emails", communicationHandler.ListEmails)
				communications.POST("/emails", communicationHandler.ComposeEmail)
				communications.GET("/emails/statistics", communicationHandler.GetEmailStatistics)
				communications.GET("/emails/search", communicationHandler.SearchEmails)
				communications.POST("/emails/preview-recipients", communicationHandler.PreviewRecipients)
				communications.POST("/emails/co-tutors", communicationHandler.SendToCoTutors)
				communications.GET("/emails/:id", communicationHandler.GetEmail)
				communications.DELETE("/emails/:id", communicationHandler.DeleteEmail)
				communications.POST("/emails/:id/schedule", communicationHandler.ScheduleEmail)
				communications.POST("/emails/:id/send", communicationHandler.SendEmailNow)
				communications.POST("/emails/:id/cancel", communicationHandler.CancelEmail)
			}

			notifications := protected.Group("/notifications")
			{
				notifications.GET("", notificationHandler.ListNotifications)
				notifications.GET("/unread-count", notificationHandler.GetUnreadCount)
				notifications.POST("/mark-all-read", notificationHandler.MarkAllAsRead)
				notifications.GET("/:id", notificationHandler.GetNotification)
				notifications.POST("/:id/read", notificationHandler.MarkAsRead)
				notifications.DELETE("/:id", notificationHandler.DeleteNotification)
			}

			billing := protected.Group("/billing")
			{
				// Invoice routes
				billing.GET("/invoices", billingHandler.ListInvoices)
				billing.GET("/invoices/upcoming", billingHandler.GetUpcomingInvoices)
				billing.GET("/invoices/overdue", billingHandler.GetOverdueInvoices)
				billing.GET("/invoices/:id", billingHandler.GetInvoice)
				billing.POST("/invoices/:id/pay", billingHandler.MarkInvoiceAsPaid)
				billing.POST("/invoices/:id/dispute", billingHandler.DisputeInvoice)
				billing.POST("/invoices/:id/void", billingHandler.VoidInvoice)

				// Billing adjustment routes
				billing.GET("/adjustments", billingHandler.ListBillingAdjustments)
				billing.POST("/adjustments", billingHandler.ApplyBillingAdjustment)

				// Subscription routes
				billing.GET("/subscriptions", billingHandler.ListSubscriptions)
				billing.GET("/subscriptions/stats", billingHandler.GetSubscriptionStatistics)
				billing.GET("/subscriptions/tenant", billingHandler.GetTenantSubscription)
				billing.GET("/subscriptions/:id", billingHandler.GetSubscription)
				billing.POST("/subscriptions/:id/cancel", billingHandler.CancelSubscription)
				billing.POST("/subscriptions/:id/reactivate", billingHandler.ReactivateSubscription)

				// Statistics and metrics
				billing.GET("/metrics", billingHandler.GetBillingMetrics)
				billing.GET("/stats", billingHandler.GetTenantBillingStats)
			}

			audit := protected.Group("/audit")
			{
				audit.GET("/logs", auditHandler.ListAuditLogs)
				audit.GET("/logs/resource", auditHandler.GetResourceAuditTrail)
				audit.GET("/logs/:id", auditHandler.GetAuditLog)
			}
		}
	}

	return router
}

func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "arc-lms-api",
		"version": "1.0.0",
	})
}

func placeholderHandler(message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": message,
			"status":  "not_implemented",
			"note":    "This endpoint structure is ready, implementation pending",
		})
	}
}
