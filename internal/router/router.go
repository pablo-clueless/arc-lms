package router

import (
	"database/sql"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"arc-lms/internal/handler"
	"arc-lms/internal/middleware"
	"arc-lms/internal/pkg/idempotency"
	"arc-lms/internal/pkg/jwt"
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

	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})

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
	router.GET("/docs/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
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
	auditRepo := postgres.NewAuditRepository(cfg.DB)

	auditService := service.NewAuditService(auditRepo)
	billingService := service.NewBillingService(nil, nil, nil, auditService) // TODO: Implement billing/invoice repository
	authService := service.NewAuthService(userRepo, cfg.JWTManager, auditService)
	userService := service.NewUserService(userRepo, auditService)
	tenantService := service.NewTenantService(tenantRepo, userRepo, auditService)
	sessionService := service.NewSessionService(sessionRepo, auditService)
	termService := service.NewTermService(termRepo, sessionRepo, enrollmentRepo, auditService, billingService)
	classService := service.NewClassService(classRepo, sessionRepo, auditService)
	courseService := service.NewCourseService(courseRepo, classRepo, userRepo, auditService)
	enrollmentService := service.NewEnrollmentService(enrollmentRepo, classRepo, userRepo, sessionRepo, auditService)
	dashboardService := service.NewDashboardService(tenantRepo, userRepo, sessionRepo, classRepo, courseRepo, enrollmentRepo)

	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	tenantHandler := handler.NewTenantHandler(tenantService)
	sessionHandler := handler.NewSessionHandler(sessionService)
	termHandler := handler.NewTermHandler(termService)
	classHandler := handler.NewClassHandler(classService)
	courseHandler := handler.NewCourseHandler(courseService)
	enrollmentHandler := handler.NewEnrollmentHandler(enrollmentService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)

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
			// Dashboard endpoint (all authenticated users)
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

			// Timetable routes (placeholders for now)
			timetables := protected.Group("/timetables")
			{
				timetables.GET("", placeholderHandler("List timetables"))
				timetables.POST("/generate", placeholderHandler("Generate timetable"))
				timetables.GET("/:id", placeholderHandler("Get timetable"))
				timetables.POST("/:id/publish", placeholderHandler("Publish timetable"))
			}

			// Assessment routes (placeholders for now)
			assessments := protected.Group("/assessments")
			{
				quizzes := assessments.Group("/quizzes")
				{
					quizzes.GET("", placeholderHandler("List quizzes"))
					quizzes.POST("", placeholderHandler("Create quiz"))
					quizzes.GET("/:id", placeholderHandler("Get quiz"))
					quizzes.PUT("/:id", placeholderHandler("Update quiz"))
					quizzes.DELETE("/:id", placeholderHandler("Delete quiz"))
					quizzes.POST("/:id/submit", placeholderHandler("Submit quiz"))
					quizzes.POST("/:id/grade", placeholderHandler("Grade quiz"))
				}

				assignments := assessments.Group("/assignments")
				{
					assignments.GET("", placeholderHandler("List assignments"))
					assignments.POST("", placeholderHandler("Create assignment"))
					assignments.GET("/:id", placeholderHandler("Get assignment"))
					assignments.PUT("/:id", placeholderHandler("Update assignment"))
					assignments.DELETE("/:id", placeholderHandler("Delete assignment"))
					assignments.POST("/:id/submit", placeholderHandler("Submit assignment"))
					assignments.POST("/:id/grade", placeholderHandler("Grade assignment"))
				}
			}

			// Examination routes (placeholders)
			examinations := protected.Group("/examinations")
			{
				examinations.GET("", placeholderHandler("List examinations"))
				examinations.POST("", placeholderHandler("Create examination"))
				examinations.GET("/:id", placeholderHandler("Get examination"))
				examinations.PUT("/:id", placeholderHandler("Update examination"))
				examinations.DELETE("/:id", placeholderHandler("Delete examination"))
				examinations.POST("/:id/start", placeholderHandler("Start examination"))
				examinations.POST("/:id/submit", placeholderHandler("Submit examination"))
				examinations.POST("/:id/publish-results", placeholderHandler("Publish results"))
			}

			// Progress routes (placeholders)
			progress := protected.Group("/progress")
			{
				progress.GET("/students/:student_id", placeholderHandler("Get student progress"))
				progress.GET("/courses/:course_id", placeholderHandler("Get course progress"))
				progress.GET("/report-cards/:student_id", placeholderHandler("Get report card"))
			}

			// Meeting routes (placeholders)
			meetings := protected.Group("/meetings")
			{
				meetings.GET("", placeholderHandler("List meetings"))
				meetings.POST("", placeholderHandler("Schedule meeting"))
				meetings.GET("/:id", placeholderHandler("Get meeting"))
				meetings.PUT("/:id", placeholderHandler("Update meeting"))
				meetings.POST("/:id/start", placeholderHandler("Start meeting"))
				meetings.POST("/:id/end", placeholderHandler("End meeting"))
				meetings.POST("/:id/cancel", placeholderHandler("Cancel meeting"))
			}

			// Communication routes (placeholders)
			communications := protected.Group("/communications")
			{
				communications.POST("/emails", placeholderHandler("Send email"))
				communications.GET("/emails", placeholderHandler("List emails"))
				communications.GET("/emails/:id", placeholderHandler("Get email"))
			}

			// Notification routes (placeholders)
			notifications := protected.Group("/notifications")
			{
				notifications.GET("", placeholderHandler("List notifications"))
				notifications.GET("/:id", placeholderHandler("Get notification"))
				notifications.POST("/:id/read", placeholderHandler("Mark as read"))
				notifications.POST("/mark-all-read", placeholderHandler("Mark all as read"))
			}

			// Billing routes (placeholders)
			billing := protected.Group("/billing")
			{
				billing.GET("/subscriptions", placeholderHandler("List subscriptions"))
				billing.GET("/invoices", placeholderHandler("List invoices"))
				billing.GET("/invoices/:id", placeholderHandler("Get invoice"))
				billing.POST("/invoices/:id/pay", placeholderHandler("Mark invoice as paid"))
			}

			// Audit routes (placeholders)
			audit := protected.Group("/audit")
			{
				audit.GET("/logs", placeholderHandler("List audit logs"))
				audit.GET("/logs/:id", placeholderHandler("Get audit log"))
			}
		}
	}

	// Suppress unused variable warnings
	_ = timetableRepo

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
