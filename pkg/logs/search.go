package logs

import (
	"context"
	"time"
)

type LogLevel int32

const (
	LogLevelUnknown  = LogLevel(0)
	LogLevelDebug    = LogLevel(1)
	LogLevelInfo     = LogLevel(2)
	LogLevelWarn     = LogLevel(3)
	LogLevelTrace    = LogLevel(4)
	LogLevelCritical = LogLevel(5)
	LogLevelError    = LogLevel(6)
)

// SearchLogClient provides the log searching service for servers
type SearchLogClient interface {
	SearchLog(ctx context.Context,
		host string, port int,
		startTime, endTime time.Time,
		levels []LogLevel, patterns []string,
		limit int) ([]string, error)
}
