package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/log"
)

func WrapErrors(errs []error) error {
	if len(errs) < 1 {
		return nil
	}
	var msgs []string
	for _, err := range errs {
		msgs = append(msgs, err.Error())
	}
	return errors.New(strings.Join(msgs, ","))
}

const (
	waitMaxRetry   = 200
	waitRetrySleep = time.Millisecond * 100
)

// CheckFunc is a condition checker that passed to WaitUntil. Its implementation
// may call c.Fatal() to abort the test, or c.Log() to add more information.
type CheckFunc func() bool

// WaitOp represents available options when execute WaitUntil
type WaitOp struct {
	retryTimes    int
	sleepInterval time.Duration
}

// WaitOption configures WaitOp
type WaitOption func(op *WaitOp)

// WithRetryTimes specify the retry times
func WithRetryTimes(retryTimes int) WaitOption {
	return func(op *WaitOp) { op.retryTimes = retryTimes }
}

// WithSleepInterval specify the sleep duration
func WithSleepInterval(sleep time.Duration) WaitOption {
	return func(op *WaitOp) { op.sleepInterval = sleep }
}

// WaitUntil repeatedly evaluates f() for a period of time, util it returns true.
func WaitUntil(f CheckFunc, opts ...WaitOption) error {
	log.Info("wait start")
	option := &WaitOp{
		retryTimes:    waitMaxRetry,
		sleepInterval: waitRetrySleep,
	}
	for _, opt := range opts {
		opt(option)
	}
	for i := 0; i < option.retryTimes; i++ {
		if f() {
			return nil
		}
		time.Sleep(option.sleepInterval)
	}
	return fmt.Errorf("wait timeout")
}
