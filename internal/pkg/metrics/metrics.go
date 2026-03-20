package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Collector tracks system metrics
type Collector struct {
	startTime       time.Time
	requestCount    atomic.Int64
	errorCount      atomic.Int64
	totalLatencyMs  atomic.Int64
	activeConns     atomic.Int64
	mu              sync.RWMutex
}

// SystemMetrics represents current system metrics
type SystemMetrics struct {
	UptimeSeconds    int64   `json:"uptime_seconds"`
	UptimeFormatted  string  `json:"uptime_formatted"`
	TotalRequests    int64   `json:"total_requests"`
	ErrorCount       int64   `json:"error_count"`
	ErrorRate        float64 `json:"error_rate"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	ActiveConns      int64   `json:"active_connections"`
}

var (
	globalCollector *Collector
	once            sync.Once
)

// GetCollector returns the global metrics collector singleton
func GetCollector() *Collector {
	once.Do(func() {
		globalCollector = &Collector{
			startTime: time.Now(),
		}
	})
	return globalCollector
}

// RecordRequest records a request with its latency and status
func (c *Collector) RecordRequest(latencyMs int64, isError bool) {
	c.requestCount.Add(1)
	c.totalLatencyMs.Add(latencyMs)
	if isError {
		c.errorCount.Add(1)
	}
}

// IncrementActiveConns increments active connection count
func (c *Collector) IncrementActiveConns() {
	c.activeConns.Add(1)
}

// DecrementActiveConns decrements active connection count
func (c *Collector) DecrementActiveConns() {
	c.activeConns.Add(-1)
}

// SetActiveConns sets the active connection count directly
func (c *Collector) SetActiveConns(count int64) {
	c.activeConns.Store(count)
}

// GetMetrics returns current system metrics
func (c *Collector) GetMetrics() *SystemMetrics {
	uptime := time.Since(c.startTime)
	totalRequests := c.requestCount.Load()
	errorCount := c.errorCount.Load()
	totalLatency := c.totalLatencyMs.Load()

	var errorRate float64
	var avgLatency float64

	if totalRequests > 0 {
		errorRate = float64(errorCount) / float64(totalRequests) * 100
		avgLatency = float64(totalLatency) / float64(totalRequests)
	}

	return &SystemMetrics{
		UptimeSeconds:   int64(uptime.Seconds()),
		UptimeFormatted: formatDuration(uptime),
		TotalRequests:   totalRequests,
		ErrorCount:      errorCount,
		ErrorRate:       errorRate,
		AvgLatencyMs:    avgLatency,
		ActiveConns:     c.activeConns.Load(),
	}
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return formatDays(days, hours, minutes)
	}
	if hours > 0 {
		return formatHours(hours, minutes, seconds)
	}
	if minutes > 0 {
		return formatMinutes(minutes, seconds)
	}
	return formatSeconds(seconds)
}

func formatDays(days, hours, minutes int) string {
	return pluralize(days, "day") + " " + pluralize(hours, "hour") + " " + pluralize(minutes, "minute")
}

func formatHours(hours, minutes, seconds int) string {
	return pluralize(hours, "hour") + " " + pluralize(minutes, "minute") + " " + pluralize(seconds, "second")
}

func formatMinutes(minutes, seconds int) string {
	return pluralize(minutes, "minute") + " " + pluralize(seconds, "second")
}

func formatSeconds(seconds int) string {
	return pluralize(seconds, "second")
}

func pluralize(count int, singular string) string {
	if count == 1 {
		return "1 " + singular
	}
	return itoa(count) + " " + singular + "s"
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(buf[pos:])
}
