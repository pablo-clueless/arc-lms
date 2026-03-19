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

	"arc-lms/internal/middleware"
	"arc-lms/internal/pkg/idempotency"
	"arc-lms/internal/pkg/jwt"
)

// Router configuration
type RouterConfig struct {
	DB              *sql.DB
	RedisClient     *redis.Client
	JWTManager      *jwt.Manager
	Environment     string
	AllowedOrigins  []string
}

// SetupRouter configures and returns the Gin router
func SetupRouter(cfg *RouterConfig) *gin.Engine {
	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Initialize logger
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})

	// Global middleware
	router.Use(middleware.RecoveryMiddleware(logger))
	router.Use(middleware.LoggerMiddleware(logger))
	router.Use(middleware.CORSMiddleware(&middleware.CORSConfig{
		AllowedOrigins: cfg.AllowedOrigins,
	}))

	// Health check endpoint (no auth required)
	router.GET("/health", healthCheckHandler)
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// API documentation
	router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/api/v1/openapi.json"),
	))

	// Serve OpenAPI spec
	router.StaticFile("/api/v1/openapi.json", "./docs/openapi/openapi.json")

	// ReDoc documentation
	router.GET("/redoc", func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		html := `<!DOCTYPE html>
<html>
<head>
	<title>Arc LMS API Documentation</title>
	<meta charset="utf-8"/>
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
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

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Public routes (no authentication required)
		public := v1.Group("/public")
		{
			// Auth endpoints will go here
			public.POST("/auth/login", placeholderHandler("Login endpoint - to be implemented"))
			public.POST("/auth/register", placeholderHandler("Register endpoint - to be implemented"))
			public.POST("/auth/password-reset", placeholderHandler("Password reset endpoint - to be implemented"))
		}

		// Protected routes (authentication required)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.JWTManager))

		// Initialize idempotency store
		idempotencyStore := idempotency.NewStore(cfg.RedisClient)

		// Apply rate limiting and idempotency to protected routes
		protected.Use(middleware.RateLimitMiddleware(cfg.RedisClient, &middleware.RateLimitConfig{
			TenantRateLimit:     1000,
			SuperAdminRateLimit: 2000,
			BurstMultiplier:     2,
			BurstWindow:         30,
			Window:              60,
		}))
		protected.Use(middleware.IdempotencyMiddleware(idempotencyStore))

		{
			// Tenant routes (SUPER_ADMIN only)
			tenants := protected.Group("/tenants")
			// tenants.Use(middleware.RequireRole("SUPER_ADMIN"))
			{
				tenants.GET("", placeholderHandler("List tenants"))
				tenants.POST("", placeholderHandler("Create tenant"))
				tenants.GET("/:id", placeholderHandler("Get tenant"))
				tenants.PUT("/:id", placeholderHandler("Update tenant"))
				tenants.DELETE("/:id", placeholderHandler("Delete tenant"))
				tenants.POST("/:id/suspend", placeholderHandler("Suspend tenant"))
				tenants.POST("/:id/reactivate", placeholderHandler("Reactivate tenant"))
			}

			// User routes
			users := protected.Group("/users")
			{
				users.GET("/me", placeholderHandler("Get current user profile"))
				users.PUT("/me", placeholderHandler("Update current user profile"))
				users.POST("/invite", placeholderHandler("Invite user"))
				users.GET("", placeholderHandler("List users"))
				users.GET("/:id", placeholderHandler("Get user"))
				users.PUT("/:id", placeholderHandler("Update user"))
				users.POST("/:id/deactivate", placeholderHandler("Deactivate user"))
			}

			// Session routes
			sessions := protected.Group("/sessions")
			{
				sessions.GET("", placeholderHandler("List sessions"))
				sessions.POST("", placeholderHandler("Create session"))
				sessions.GET("/:id", placeholderHandler("Get session"))
				sessions.PUT("/:id", placeholderHandler("Update session"))
				sessions.DELETE("/:id", placeholderHandler("Delete session"))
				sessions.POST("/:id/activate", placeholderHandler("Activate session"))
			}

			// Term routes
			terms := protected.Group("/sessions/:session_id/terms")
			{
				terms.GET("", placeholderHandler("List terms"))
				terms.POST("", placeholderHandler("Create term"))
				terms.GET("/:id", placeholderHandler("Get term"))
				terms.PUT("/:id", placeholderHandler("Update term"))
				terms.DELETE("/:id", placeholderHandler("Delete term"))
				terms.POST("/:id/activate", placeholderHandler("Activate term"))
			}

			// Class routes
			classes := protected.Group("/classes")
			{
				classes.GET("", placeholderHandler("List classes"))
				classes.POST("", placeholderHandler("Create class"))
				classes.GET("/:id", placeholderHandler("Get class"))
				classes.PUT("/:id", placeholderHandler("Update class"))
				classes.DELETE("/:id", placeholderHandler("Delete class"))
			}

			// Course routes
			courses := protected.Group("/courses")
			{
				courses.GET("", placeholderHandler("List courses"))
				courses.POST("", placeholderHandler("Create course"))
				courses.GET("/:id", placeholderHandler("Get course"))
				courses.PUT("/:id", placeholderHandler("Update course"))
				courses.DELETE("/:id", placeholderHandler("Delete course"))
			}

			// Timetable routes
			timetables := protected.Group("/timetables")
			{
				timetables.GET("", placeholderHandler("List timetables"))
				timetables.POST("/generate", placeholderHandler("Generate timetable"))
				timetables.GET("/:id", placeholderHandler("Get timetable"))
				timetables.POST("/:id/publish", placeholderHandler("Publish timetable"))
			}

			// Assessment routes (Quizzes and Assignments)
			assessments := protected.Group("/assessments")
			{
				// Quizzes
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

				// Assignments
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

			// Examination routes
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

			// Progress and Report Cards
			progress := protected.Group("/progress")
			{
				progress.GET("/students/:student_id", placeholderHandler("Get student progress"))
				progress.GET("/courses/:course_id", placeholderHandler("Get course progress"))
				progress.GET("/report-cards/:student_id", placeholderHandler("Get report card"))
			}

			// Meeting routes
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

			// Communication routes
			communications := protected.Group("/communications")
			{
				communications.POST("/emails", placeholderHandler("Send email"))
				communications.GET("/emails", placeholderHandler("List emails"))
				communications.GET("/emails/:id", placeholderHandler("Get email"))
			}

			// Notification routes
			notifications := protected.Group("/notifications")
			{
				notifications.GET("", placeholderHandler("List notifications"))
				notifications.GET("/:id", placeholderHandler("Get notification"))
				notifications.POST("/:id/read", placeholderHandler("Mark as read"))
				notifications.POST("/mark-all-read", placeholderHandler("Mark all as read"))
			}

			// Billing routes
			billing := protected.Group("/billing")
			{
				billing.GET("/subscriptions", placeholderHandler("List subscriptions"))
				billing.GET("/invoices", placeholderHandler("List invoices"))
				billing.GET("/invoices/:id", placeholderHandler("Get invoice"))
				billing.POST("/invoices/:id/pay", placeholderHandler("Mark invoice as paid"))
			}

			// Audit routes
			audit := protected.Group("/audit")
			{
				audit.GET("/logs", placeholderHandler("List audit logs"))
				audit.GET("/logs/:id", placeholderHandler("Get audit log"))
			}
		}
	}

	return router
}

// healthCheckHandler returns API health status
func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "arc-lms-api",
		"version": "1.0.0",
	})
}

// placeholderHandler returns a placeholder response for unimplemented endpoints
func placeholderHandler(message string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": message,
			"status":  "not_implemented",
			"note":    "This endpoint structure is ready, implementation pending",
		})
	}
}
