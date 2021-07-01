package logsearch

import (
	"context"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/pingcap/kvproto/pkg/diagnosticspb"
	"google.golang.org/grpc"

	"github.com/pingcap/tipocket/pkg/logs"
)

var (
	_ logs.SearchLogClient = &DiagnosticLogClient{}
)

// DiagnosticLogClient is a logs.SearchLogClient implementation
type DiagnosticLogClient struct{}

// NewDiagnosticLogClient creates an DiagnosticLogClient instance
func NewDiagnosticLogClient() *DiagnosticLogClient {
	return &DiagnosticLogClient{}
}

// SearchLog ...
func (d *DiagnosticLogClient) SearchLog(ctx context.Context,
	host string, port int,
	startTime, endTime time.Time,
	levels []logs.LogLevel, patterns []string,
	limit int) (logs []string, err error) {
	logClient, err := d.connect(host, port)
	if err != nil {
		return
	}
	logLevels := make([]diagnosticspb.LogLevel, 0, len(levels))
	for _, level := range levels {
		logLevels = append(logLevels, diagnosticspb.LogLevel(level))
	}
	stream, err := logClient.SearchLog(ctx,
		&diagnosticspb.SearchLogRequest{
			StartTime: startTime.Unix() * 1000,
			EndTime:   endTime.Unix() * 1000,
			Levels:    logLevels,
			Patterns:  patterns,
		},
	)
	if err != nil {
		return
	}
Scanlog:
	for {
		var res *diagnosticspb.SearchLogResponse
		res, err = stream.Recv()
		if err != nil {
			if err != io.EOF {
				return
			}
			err = nil
			break Scanlog
		}
		for _, msg := range res.Messages {
			timeStr := time.Unix(0, msg.Time*int64(time.Millisecond)).Format("2006/01/02 15:04:05.000 -07:00")
			logs = append(logs, fmt.Sprintf("[%s] [%s] %s\n", timeStr, msg.Level.String(), msg.Message))
			if len(logs) >= limit {
				break Scanlog
			}
		}
	}
	return
}

func (d *DiagnosticLogClient) connect(host string, port int) (diagnosticspb.DiagnosticsClient, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", host, port),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(math.MaxInt64)),
	)
	if err != nil {
		return nil, err
	}
	return diagnosticspb.NewDiagnosticsClient(conn), nil
}
