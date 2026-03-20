package config

import (
	"crypto/tls"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	SMTP     SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

type ServerConfig struct {
	Port         string
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addr      string
	Password  string
	DB        int
	TLSConfig *tls.Config
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration // 15 minutes
	RefreshTTL    time.Duration // 30 days
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			Environment:  getEnv("GO_ENV", "development"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/lms?sslmode=disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 25),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: parseRedisConfig(),
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", ""),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", ""),
			AccessTTL:     getDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:    getDurationEnv("JWT_REFRESH_TTL", 30*24*time.Hour),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getIntEnv("SMTP_PORT", 587),
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@example.com"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if c.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT_REFRESH_SECRET is required")
	}
	if c.Database.URL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	return nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// parseRedisConfig parses Redis configuration from REDIS_URL or individual env vars
func parseRedisConfig() RedisConfig {
	// Check if REDIS_URL is provided
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		return parseRedisURL(redisURL)
	}

	// Fall back to individual environment variables
	return RedisConfig{
		Addr:      getEnv("REDIS_ADDR", "localhost:6379"),
		Password:  getEnv("REDIS_PASSWORD", ""),
		DB:        getIntEnv("REDIS_DB", 0),
		TLSConfig: nil,
	}
}

// parseRedisURL parses a Redis URL (redis:// or rediss://)
func parseRedisURL(redisURL string) RedisConfig {
	parsedURL, err := url.Parse(redisURL)
	if err != nil {
		// Fall back to defaults if parsing fails
		return RedisConfig{
			Addr:      "localhost:6379",
			Password:  "",
			DB:        0,
			TLSConfig: nil,
		}
	}

	config := RedisConfig{
		Addr:      parsedURL.Host,
		Password:  "",
		DB:        0,
		TLSConfig: nil,
	}

	// Extract password from URL
	if parsedURL.User != nil {
		password, _ := parsedURL.User.Password()
		config.Password = password
	}

	// Extract database number from path
	if len(parsedURL.Path) > 1 {
		dbStr := parsedURL.Path[1:] // Remove leading "/"
		if db, err := strconv.Atoi(dbStr); err == nil {
			config.DB = db
		}
	}

	// Enable TLS for rediss:// scheme
	if parsedURL.Scheme == "rediss" {
		config.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return config
}
