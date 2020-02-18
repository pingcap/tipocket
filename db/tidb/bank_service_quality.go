package tidb

import (
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/pingcap/tipocket/pkg/core"
)

type rtoMetrics struct {
	key string
	p80 time.Duration
	p90 time.Duration
	p99 time.Duration
}

type bankServiceQualityChecker struct {
	profFilePath string
}

type countWindow struct {
	series []time.Time
	pos    int
}

func (f *countWindow) count(duration time.Duration) (*time.Time, int, error) {
	if f.pos >= len(f.series) {
		return &f.series[len(f.series)-1], 0, fmt.Errorf("window has ended")
	}

	var count = 0
	var pos = f.pos
	var startTime = f.series[pos]

	defer func() {
		f.pos++
	}()

	for pos < len(f.series) {
		t := f.series[pos]
		if t.After(startTime.Add(duration)) {
			break
		}
		count++
		pos++
	}
	return &startTime, count, nil
}

// Check checks the bank history.
func (c bankServiceQualityChecker) Check(_ core.Model, ops []core.Operation) (bool, error) {
	var key string
	var duration = time.Minute
	var failCount float64
	var failCountPerMinuteBeforeInject float64

	var injectionTime time.Time
	//var recoveryTime time.Time

	var p80 *time.Duration
	var p90 *time.Duration
	var p99 *time.Duration

	startTime := ops[0].Time
	idx := 0
	for idx < len(ops) {
		op := ops[idx]
		if op.Action == core.ReturnOperation {
			response := op.Data.(bankResponse)
			if response.Unknown {
				failCount++
			}
		} else if op.Action == core.InvokeNemesis {
			injectionTime = op.Time
			key = op.Data.(string)
			break
		} else if op.Action == core.RecoverNemesis {
			//recoveryTime = op.Time
			//break
		}
		idx++
	}
	failCountPerMinuteBeforeInject = failCount / injectionTime.Sub(startTime).Minutes()
	if idx == len(ops) {
		log.Panic("expect more events after recovery from nemesis")
	}

	window := convertToFailSeries(ops[idx:])

	for true {
		startTime, failCount, err := window.count(duration)
		dur := startTime.Sub(injectionTime)
		if err != nil {
			if p80 == nil {
				p80 = &dur
			}
			if p90 == nil {
				p90 = &dur
			}
			if p99 == nil {
				p99 = &dur
			}
			break
		}
		failCountPerMinute := float64(failCount) / duration.Minutes()

		if 0.8*failCountPerMinute <= failCountPerMinuteBeforeInject && p80 == nil {
			p80 = &dur
		}
		if 0.9*failCountPerMinute <= failCountPerMinuteBeforeInject && p90 == nil {
			p90 = &dur
		}
		if 0.99*failCountPerMinute <= failCountPerMinuteBeforeInject && p99 == nil {
			p99 = &dur
			break
		}
	}
	metrics := rtoMetrics{key: key}
	if p80 != nil {
		metrics.p80 = *p80
	}
	if p90 != nil {
		metrics.p90 = *p90
	}
	if p99 != nil {
		metrics.p99 = *p99
	}

	if err := c.record(&metrics); err != nil {
		return false, err
	}
	return true, nil
}

func (c bankServiceQualityChecker) record(rto *rtoMetrics) error {
	os.MkdirAll(path.Dir(""), 0755)
	f, err := os.OpenFile(c.profFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString(fmt.Sprintf("key:%s,p80:%.2f,p90:%.2f,p99:%.2f\n", rto.key, rto.p80.Seconds(), rto.p90.Seconds(), rto.p99.Seconds())); err != nil {
		return err
	}
	return nil
}

func convertToFailSeries(ops []core.Operation) *countWindow {
	var series []time.Time
	for idx := range ops {
		op := ops[idx]
		if op.Action == core.ReturnOperation {
			response, ok := op.Data.(bankResponse)
			if !ok {
				continue
			}
			if response.Unknown {
				if !op.Time.IsZero() {
					series = append(series, op.Time)
				}
			}
		}
	}
	return &countWindow{
		series: series,
		pos:    0,
	}
}

// Name returns the name of the verifier.
func (bankServiceQualityChecker) Name() string {
	return "tidb_bank_service_quality_checker"
}

// BankServiceQualityChecker checks the bank history and analyze service quality
func BankServiceQualityChecker(path string) core.Checker {
	return bankServiceQualityChecker{profFilePath: path}
}
