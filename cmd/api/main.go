package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"arc-lms/internal/config"
	"arc-lms/internal/pkg/jwt"
	"arc-lms/internal/router"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Arc LMS API in %s mode", cfg.Server.Environment)

	// Initialize database connection
	db, err := initDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("✓ Database connection established")

	// Initialize Redis connection
	redisClient, err := initRedis(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redisClient.Close()

	// Test Redis connection (use longer timeout for remote connections)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("✓ Redis connection established")

	// Initialize JWT manager
	jwtManager := jwt.NewManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTL,
		cfg.JWT.RefreshTTL,
	)

	log.Println("✓ JWT manager initialized")

	// Set up router
	allowedOrigins := []string{"http://localhost:3000", "http://localhost:8080"}
	if cfg.Server.Environment == "production" {
		allowedOrigins = []string{"https://arc-lms.onrender.com"}
	}

	r := router.SetupRouter(&router.RouterConfig{
		DB:             db,
		RedisClient:    redisClient,
		JWTManager:     jwtManager,
		Environment:    cfg.Server.Environment,
		AllowedOrigins: allowedOrigins,
	})

	log.Println("✓ Router configured")

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("✓ Server starting on port %s", cfg.Server.Port)
		log.Printf("✓ Swagger UI: http://localhost:%s/docs/index.html", cfg.Server.Port)
		log.Printf("✓ ReDoc: http://localhost:%s/redoc", cfg.Server.Port)
		log.Printf("✓ Health check: http://localhost:%s/health", cfg.Server.Port)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✓ Server exited gracefully")
}

// initDatabase initializes and returns a database connection
func initDatabase(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// initRedis initializes and returns a Redis client
func initRedis(cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		TLSConfig:    cfg.TLSConfig,
		DialTimeout:  10 * time.Second,  // Time to establish connection
		ReadTimeout:  10 * time.Second,  // Time to read response
		WriteTimeout: 10 * time.Second,  // Time to write request
		PoolSize:     10,                 // Connection pool size
		MinIdleConns: 5,                  // Minimum idle connections
	})

	return client, nil
}
