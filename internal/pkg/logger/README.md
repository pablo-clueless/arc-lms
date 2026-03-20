# Colored Logger

This package provides colored, human-readable logging for the Arc LMS API.

## Features

- **Color-coded log levels**:
  - 🔵 **BLUE** - INFO logs (successful operations)
  - 🟡 **YELLOW** - WARN logs (client errors, warnings)
  - 🔴 **RED** - ERROR logs (server errors, panics)
  - 🟢 **GREEN** - DEBUG logs (debug information)

- **Contextual coloring**:
  - HTTP methods: GET (blue), POST (green), PUT/PATCH (yellow), DELETE (red)
  - Status codes: 2xx (green), 3xx (blue), 4xx (yellow), 5xx (red)
  - Latency: Fast <500ms (green), Medium 500-1000ms (yellow), Slow >1000ms (red)

- **Structured fields**: Important fields like `request_id`, `method`, `path`, `status`, `latency_ms` are displayed first

- **Environment-aware**:
  - Development: Colored, human-readable output
  - Production: JSON format for log aggregation tools

## Log Format

```
2026-03-20 15:04:05 [INFO ] Request completed request_id=abc-123 method=GET path=/api/v1/users status=200 latency_ms=15ms ip=127.0.0.1 user_id=uuid role=ADMIN
```

## Example Output (with colors)

### Successful Request (INFO - Blue)
```
2026-03-20 15:04:05 [INFO ] Request completed request_id=abc-123 method=GET path=/api/v1/dashboard status=200 latency_ms=45ms ip=127.0.0.1 user_id=550e8400-e29b-41d4-a716-446655440000 tenant_id=660e8400-e29b-41d4-a716-446655440000 role=ADMIN
```

### Client Error (WARN - Yellow)
```
2026-03-20 15:04:07 [WARN ] Client error request_id=def-456 method=POST path=/api/v1/users status=400 latency_ms=12ms ip=127.0.0.1 errors=validation failed
```

### Server Error (ERROR - Red)
```
2026-03-20 15:04:10 [ERROR] Internal server error request_id=ghi-789 method=POST path=/api/v1/courses status=500 latency_ms=234ms ip=127.0.0.1 user_id=550e8400-e29b-41d4-a716-446655440000 tenant_id=660e8400-e29b-41d4-a716-446655440000 role=TUTOR errors=database connection failed
```

### Panic Recovery (ERROR - Red)
```
2026-03-20 15:04:12 [ERROR] Panic recovered in request handler request_id=jkl-012 method=GET path=/api/v1/enrollments status=500 latency_ms=5ms ip=127.0.0.1 panic=runtime error: invalid memory address or nil pointer dereference stack_trace=...
```

## Usage

The logger is automatically configured in the router based on the environment:

```go
// In router/router.go
var logger *logrus.Logger
if cfg.Environment == "production" {
    logger = customLogger.NewJSONLogger()  // JSON format for production
} else {
    logger = customLogger.NewColoredLogger()  // Colored format for development
}
```

### Manual Usage

You can also create loggers manually:

```go
import customLogger "arc-lms/internal/pkg/logger"

// Create colored logger
logger := customLogger.NewColoredLogger()

// Create JSON logger
logger := customLogger.NewJSONLogger()

// Use the logger
logger.Info("Application started")
logger.WithFields(logrus.Fields{
    "port": 8080,
    "env": "development",
}).Info("Server listening")
```

## Configuration

The `ColoredFormatter` supports the following options:

```go
formatter := &logger.ColoredFormatter{
    TimestampFormat: "2006-01-02 15:04:05",  // Timestamp format
    DisableColors:   false,                   // Set to true to disable colors
    ShowFullLevel:   false,                   // Set to true to show full level names (INFO instead of INFO)
}
```

## Color Reference

| Element | Color | ANSI Code |
|---------|-------|-----------|
| INFO | Blue (Bold) | `\033[1;34m` |
| WARN | Yellow (Bold) | `\033[1;33m` |
| ERROR | Red (Bold) | `\033[1;31m` |
| DEBUG | Green (Bold) | `\033[1;32m` |
| Timestamp | Gray | `\033[90m` |
| Fields | Cyan | `\033[36m` |
| Status 2xx | Green | `\033[32m` |
| Status 3xx | Blue | `\033[34m` |
| Status 4xx | Yellow | `\033[33m` |
| Status 5xx | Red | `\033[31m` |

## Benefits

1. **Easier Debugging**: Colors make it easy to spot errors and warnings at a glance
2. **Better Development Experience**: Human-readable format during development
3. **Production Ready**: Automatically switches to JSON format in production for log aggregation
4. **Consistent**: All logs follow the same format with important fields prioritized
5. **Secure**: Sensitive data is automatically masked by the logger middleware
