package seed

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/crypto"
	"arc-lms/internal/repository"
	"arc-lms/internal/repository/postgres"

	"github.com/google/uuid"
)

// SuperAdminConfig holds superadmin seed configuration
type SuperAdminConfig struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

// SeedSuperAdmin creates a superadmin user if it doesn't exist
func SeedSuperAdmin(db *sql.DB, config SuperAdminConfig) error {
	ctx := context.Background()

	// Initialize user repository
	userRepo := postgres.NewUserRepository(db)

	// Check if superadmin already exists
	existingUser, err := userRepo.GetByEmail(ctx, config.Email)
	if err != nil && err != repository.ErrNotFound {
		return fmt.Errorf("failed to check if superadmin exists: %w", err)
	}

	// If user exists, skip seeding
	if existingUser != nil {
		log.Printf("✓ SuperAdmin already exists: %s (skipping seed)", config.Email)
		return nil
	}

	// Hash password
	hashedPassword, err := crypto.HashPassword(config.Password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create superadmin user
	superAdmin := &domain.User{
		ID:           uuid.New(),
		TenantID:     nil, // SuperAdmin has no tenant
		Role:         domain.RoleSuperAdmin,
		Email:        config.Email,
		PasswordHash: hashedPassword,
		FirstName:    config.FirstName,
		LastName:     config.LastName,
		Status:       domain.UserStatusActive,
		Permissions:  []string{"*:*"}, // Full system access
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Create user in database
	if err := userRepo.Create(ctx, superAdmin); err != nil {
		return fmt.Errorf("failed to create superadmin: %w", err)
	}

	log.Printf("✓ SuperAdmin created successfully: %s", config.Email)
	log.Printf("  - Role: SUPER_ADMIN")
	log.Printf("  - Status: ACTIVE")
	log.Printf("  - Permissions: Full system access")

	return nil
}

// SeedAll runs all seed functions
func SeedAll(db *sql.DB) error {
	log.Println("🌱 Running database seeds...")

	// Seed SuperAdmin (with default values if env vars not set)
	superAdminConfig := SuperAdminConfig{
		Email:     getEnvOrDefault("SUPERADMIN_EMAIL", "smsnmicheal@gmail.com"),
		Password:  getEnvOrDefault("SUPERADMIN_PASSWORD", "Asdfgh123@"),
		FirstName: getEnvOrDefault("SUPERADMIN_FIRST_NAME", "Super"),
		LastName:  getEnvOrDefault("SUPERADMIN_LAST_NAME", "Admin"),
	}

	if err := SeedSuperAdmin(db, superAdminConfig); err != nil {
		return fmt.Errorf("failed to seed superadmin: %w", err)
	}

	log.Println("✓ Database seeding completed")
	return nil
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
