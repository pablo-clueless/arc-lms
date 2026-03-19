package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"arc-lms/internal/domain"
	"arc-lms/internal/pkg/errors"
)

const (
	// Rate limit defaults
	defaultTenantRateLimit    = 1000      // requests per minute
	superAdminRateLimit       = 2000      // requests per minute
	burstMultiplier           = 2         // 2x sustained rate
	burstWindow               = 30        // seconds
	rateLimitWindow           = 60        // seconds (1 minute)
	rateLimitKeyPrefix        = "ratelimit"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	TenantRateLimit    int
	SuperAdminRateLimit int
	BurstMultiplier    int
	BurstWindow        int
	Window             int
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		TenantRateLimit:    defaultTenantRateLimit,
		SuperAdminRateLimit: superAdminRateLimit,
		BurstMultiplier:    burstMultiplier,
		BurstWindow:        burstWindow,
		Window:             rateLimitWindow,
	}
}

// RateLimitMiddleware implements sliding window rate limiting using Redis
func RateLimitMiddleware(redisClient *redis.Client, config *RateLimitConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Determine rate limit key and limit based on user role
		var rateLimitKey string
		var limit int

		// Extract role and tenant/user ID from context
		roleValue, roleExists := c.Get(ContextKeyRole)
		if !roleExists {
			// If no role (shouldn't happen after AuthMiddleware), allow the request
			c.Next()
			return
		}

		userRole := domain.Role(roleValue.(string))

		// Extract user_id from context
		userIDValue, userIDExists := c.Get(ContextKeyUserID)
		if !userIDExists {
			// If no user_id, allow the request
			c.Next()
			return
		}
		userID := userIDValue.(uuid.UUID)

		// Set rate limit based on role
		if userRole == domain.RoleSuperAdmin {
			// SUPER_ADMIN: rate limit per user
			rateLimitKey = fmt.Sprintf("%s:user:%s", rateLimitKeyPrefix, userID.String())
			limit = config.SuperAdminRateLimit
		} else {
			// Regular users: rate limit per tenant
			tenantIDValue, tenantIDExists := c.Get(ContextKeyTenantID)
			if !tenantIDExists {
				// If no tenant_id, allow the request
				c.Next()
				return
			}
			tenantID := tenantIDValue.(uuid.UUID)
			rateLimitKey = fmt.Sprintf("%s:tenant:%s", rateLimitKeyPrefix, tenantID.String())
			limit = config.TenantRateLimit
		}

		// Check rate limit using sliding window algorithm
		allowed, remaining, resetTime, err := checkRateLimit(ctx, redisClient, rateLimitKey, limit, config.Window, config.BurstMultiplier, config.BurstWindow)
		if err != nil {
			// Log error but allow the request (fail open)
			c.Next()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))

		if !allowed {
			// Rate limit exceeded
			retryAfter := int(time.Until(resetTime).Seconds())
			if retryAfter < 0 {
				retryAfter = 0
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))

			errors.RespondWithError(c, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED",
				"Rate limit exceeded. Please try again later.", map[string]interface{}{
					"limit":       limit,
					"retry_after": retryAfter,
				})
			c.Abort()
			return
		}

		c.Next()
	}
}

// checkRateLimit implements sliding window rate limiting with burst capacity
// Returns: allowed, remaining, resetTime, error
func checkRateLimit(ctx context.Context, redisClient *redis.Client, key string, limit int, window int, burstMultiplier int, burstWindow int) (bool, int, time.Time, error) {
	now := time.Now()
	windowStart := now.Add(-time.Duration(window) * time.Second)
	burstWindowStart := now.Add(-time.Duration(burstWindow) * time.Second)

	// Use sorted set to store request timestamps
	pipe := redisClient.Pipeline()

	// Remove old entries outside the window
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart.UnixNano()))

	// Count requests in the current window
	countCmd := pipe.ZCount(ctx, key, fmt.Sprintf("%d", windowStart.UnixNano()), "+inf")

	// Count requests in the burst window
	burstCountCmd := pipe.ZCount(ctx, key, fmt.Sprintf("%d", burstWindowStart.UnixNano()), "+inf")

	// Add current request timestamp
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	})

	// Set expiry for the key (cleanup)
	pipe.Expire(ctx, key, time.Duration(window+60)*time.Second)

	// Execute pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, 0, now, fmt.Errorf("failed to execute rate limit pipeline: %w", err)
	}

	// Get counts
	count, err := countCmd.Result()
	if err != nil {
		return false, 0, now, fmt.Errorf("failed to get request count: %w", err)
	}

	burstCount, err := burstCountCmd.Result()
	if err != nil {
		return false, 0, now, fmt.Errorf("failed to get burst count: %w", err)
	}

	// Calculate burst limit
	burstLimit := limit * burstMultiplier

	// Check if within limits
	// Allow burst capacity for short periods
	allowed := false
	remaining := 0

	if burstCount <= int64(burstLimit) {
		// Within burst limit
		allowed = true
		remaining = burstLimit - int(burstCount)
	} else if count <= int64(limit) {
		// Burst exceeded but within sustained rate
		allowed = true
		remaining = limit - int(count)
	}

	// Calculate reset time (when the window resets)
	resetTime := now.Add(time.Duration(window) * time.Second)

	return allowed, remaining, resetTime, nil
}
