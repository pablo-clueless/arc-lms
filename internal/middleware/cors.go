package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowCredentials bool
	MaxAge           int // Preflight cache duration in seconds
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowedOrigins:   []string{"*"}, // In production, specify exact origins
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS)
func CORSMiddleware(config *CORSConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCORSConfig()
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowedOrigin := ""
		if len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*" {
			// Allow all origins
			allowedOrigin = "*"
		} else {
			// Check if origin is in allowed list
			for _, allowed := range config.AllowedOrigins {
				if allowed == origin {
					allowedOrigin = origin
					break
				}
			}
		}

		// Set CORS headers if origin is allowed
		if allowedOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowedOrigin)

			// Allow credentials if configured (cannot use with wildcard origin)
			if config.AllowCredentials && allowedOrigin != "*" {
				c.Header("Access-Control-Allow-Credentials", "true")
			}

			// Expose custom headers to the client
			c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-RateLimit-Remaining, X-RateLimit-Limit, X-RateLimit-Reset, Content-Length")

			// Handle preflight requests (OPTIONS)
			if c.Request.Method == "OPTIONS" {
				// Allow specific methods
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

				// Allow specific headers
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-ID, Accept, Origin")

				// Set preflight cache duration
				c.Header("Access-Control-Max-Age", string(rune(config.MaxAge)))

				// Return 204 No Content for preflight requests
				c.AbortWithStatus(204)
				return
			}
		}

		c.Next()
	}
}

// StrictCORSMiddleware provides CORS with stricter origin validation
func StrictCORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is in allowed list
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if !allowed && origin != "" {
			// Origin not allowed, reject the request
			c.AbortWithStatus(403)
			return
		}

		if origin != "" {
			// Set CORS headers
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Expose-Headers", "X-Request-ID, X-RateLimit-Remaining, X-RateLimit-Limit, X-RateLimit-Reset, Content-Length")

			// Handle preflight requests
			if c.Request.Method == "OPTIONS" {
				c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-ID, Accept, Origin")
				c.Header("Access-Control-Max-Age", "86400")
				c.AbortWithStatus(204)
				return
			}
		}

		c.Next()
	}
}
