package main

import (
	. "net/url"
	"time"

	"go.uber.org/zap"

	. "github.com/tipocket/testcase/sdk-ticase-4885"
)

func main() {
	stopper := make(chan int)

	config := zap.NewDevelopmentConfig()
	config.DisableStacktrace = true
	logger := Try(config.Build()).(*zap.Logger)

	ListenMetrics(*Try(Parse("http://127.0.0.1:9090")).(*URL), 5*time.Second, logger, stopper)

	go func() {
		time.Sleep(5 * time.Minute)
		stopper <- 0
	}()
}
