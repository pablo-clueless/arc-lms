package logger

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorGray   = "\033[90m"

	ColorBoldRed    = "\033[1;31m"
	ColorBoldGreen  = "\033[1;32m"
	ColorBoldYellow = "\033[1;33m"
	ColorBoldBlue   = "\033[1;34m"
)

type ColoredFormatter struct {
	TimestampFormat string

	DisableColors bool

	ShowFullLevel bool
}

func (f *ColoredFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var levelColor string
	var levelText string

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = "2006-01-02 15:04:05"
	}

	if !f.DisableColors {
		switch entry.Level {
		case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
			levelColor = ColorBoldRed
			levelText = "ERROR"
		case logrus.WarnLevel:
			levelColor = ColorBoldYellow
			levelText = "WARN"
		case logrus.InfoLevel:
			levelColor = ColorBoldBlue
			levelText = "INFO"
		case logrus.DebugLevel, logrus.TraceLevel:
			levelColor = ColorBoldGreen
			levelText = "DEBUG"
		default:
			levelColor = ColorWhite
			levelText = "UNKNOWN"
		}
	} else {
		switch entry.Level {
		case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
			levelText = "ERROR"
		case logrus.WarnLevel:
			levelText = "WARN"
		case logrus.InfoLevel:
			levelText = "INFO"
		case logrus.DebugLevel, logrus.TraceLevel:
			levelText = "DEBUG"
		default:
			levelText = "UNKNOWN"
		}
	}

	if !f.ShowFullLevel && len(levelText) > 4 {
		levelText = levelText[:4]
	}

	var b strings.Builder

	timestamp := entry.Time.Format(timestampFormat)
	if !f.DisableColors {
		b.WriteString(ColorGray)
		b.WriteString(timestamp)
		b.WriteString(ColorReset)
	} else {
		b.WriteString(timestamp)
	}

	b.WriteString(" ")
	if !f.DisableColors {
		b.WriteString(levelColor)
		b.WriteString(fmt.Sprintf("[%-5s]", levelText))
		b.WriteString(ColorReset)
	} else {
		b.WriteString(fmt.Sprintf("[%-5s]", levelText))
	}

	b.WriteString(" ")
	b.WriteString(entry.Message)

	if len(entry.Data) > 0 {
		b.WriteString(" ")
		if !f.DisableColors {
			b.WriteString(ColorCyan)
		}

		importantFields := []string{"request_id", "method", "path", "status", "latency_ms", "ip", "user_id", "tenant_id", "role"}
		fieldsPrinted := make(map[string]bool)

		for _, fieldName := range importantFields {
			if value, exists := entry.Data[fieldName]; exists {
				f.writeField(&b, fieldName, value)
				fieldsPrinted[fieldName] = true
			}
		}

		for fieldName, value := range entry.Data {
			if !fieldsPrinted[fieldName] {
				f.writeField(&b, fieldName, value)
			}
		}

		if !f.DisableColors {
			b.WriteString(ColorReset)
		}
	}

	b.WriteString("\n")

	return []byte(b.String()), nil
}

func (f *ColoredFormatter) writeField(b *strings.Builder, key string, value interface{}) {

	switch key {
	case "status":
		if statusCode, ok := value.(int); ok {
			var statusColor string
			if !f.DisableColors {
				if statusCode >= 500 {
					statusColor = ColorRed
				} else if statusCode >= 400 {
					statusColor = ColorYellow
				} else if statusCode >= 300 {
					statusColor = ColorBlue
				} else {
					statusColor = ColorGreen
				}
				b.WriteString(fmt.Sprintf("%s=%s%d%s ", key, statusColor, statusCode, ColorCyan))
			} else {
				b.WriteString(fmt.Sprintf("%s=%d ", key, statusCode))
			}
			return
		}
	case "latency_ms":
		if latency, ok := value.(int64); ok {
			var latencyColor string
			if !f.DisableColors {
				if latency > 1000 {
					latencyColor = ColorRed
				} else if latency > 500 {
					latencyColor = ColorYellow
				} else {
					latencyColor = ColorGreen
				}
				b.WriteString(fmt.Sprintf("%s=%s%dms%s ", key, latencyColor, latency, ColorCyan))
			} else {
				b.WriteString(fmt.Sprintf("%s=%dms ", key, latency))
			}
			return
		}
	case "method":
		if method, ok := value.(string); ok {
			var methodColor string
			if !f.DisableColors {
				switch method {
				case "GET":
					methodColor = ColorBlue
				case "POST":
					methodColor = ColorGreen
				case "PUT", "PATCH":
					methodColor = ColorYellow
				case "DELETE":
					methodColor = ColorRed
				default:
					methodColor = ColorWhite
				}
				b.WriteString(fmt.Sprintf("%s=%s%s%s ", key, methodColor, method, ColorCyan))
			} else {
				b.WriteString(fmt.Sprintf("%s=%s ", key, method))
			}
			return
		}
	}

	b.WriteString(fmt.Sprintf("%s=%v ", key, value))
}

func NewColoredLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&ColoredFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		DisableColors:   false,
		ShowFullLevel:   false,
	})
	logger.SetLevel(logrus.InfoLevel)
	return logger
}

func NewJSONLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	logger.SetLevel(logrus.InfoLevel)
	return logger
}
