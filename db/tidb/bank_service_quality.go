package tidb

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"github.com/pingcap/tipocket/pkg/core"
)

type rtoMetrics struct {
	nemesis          *core.NemesisGeneratorRecord
	p80              time.Duration
	p90              time.Duration
	p99              time.Duration
	totalMeasureTime time.Duration
}

type bankServiceQualityChecker struct {
	profFile string
}

type countWindow struct {
	series []time.Time
	pos    int
	start  bool
}

// Next move to next pos
func (f *countWindow) Next() bool {
	if !f.start {
		f.start = true
		return f.pos < len(f.series)
	}
	f.pos++
	return f.pos < len(f.series)
}

// Count counts failure number in continuous minutes from f.pos
func (f *countWindow) Count(duration time.Duration) (*time.Time, int) {
	var count = 0
	var pos = f.pos
	var startTime = f.series[pos]

	for pos < len(f.series) {
		t := f.series[pos]
		if t.After(startTime.Add(duration)) {
			break
		}
		count++
		pos++
	}
	return &startTime, count
}

// Check checks the bank history and measure failure requests count before and after nemesis was injected.
// When the avg failure count of one continuous minute is below 0.8 * the failure count before nemesis, we say the the system
// recover to p80 line from that time point. And p90 or p99 are similar.
func (c bankServiceQualityChecker) Check(_ core.Model, ops []core.Operation) (bool, error) {
	var nemesisRecord core.NemesisGeneratorRecord
	var duration = time.Minute
	var failCount float64
	var failCountPerMinuteBeforeInject float64
	var injectionTime time.Time
	//var recoveryTime time.Time
	metrics := rtoMetrics{nemesis: &nemesisRecord}

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
			nemesisRecord = op.Data.(core.NemesisGeneratorRecord)
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

	var window countWindow

	window, metrics.totalMeasureTime = convertToFailSeries(ops[idx:])
	if len(window.series) == 0 {
		metrics.p80, metrics.p90, metrics.p99 = time.Duration(0), time.Duration(0), time.Duration(0)
	} else {
		lastFailureDuration := window.series[len(window.series)-1].Sub(injectionTime)
		metrics.p80, metrics.p90, metrics.p99 = lastFailureDuration, lastFailureDuration, lastFailureDuration
	}
	for window.Next() {
		startTime, failCount := window.Count(duration)
		dur := startTime.Sub(injectionTime)
		failCountPerMinute := float64(failCount) / duration.Minutes()

		if 0.8*failCountPerMinute <= failCountPerMinuteBeforeInject && dur < metrics.p80 {
			metrics.p80 = dur
		}
		if 0.9*failCountPerMinute <= failCountPerMinuteBeforeInject && dur < metrics.p90 {
			metrics.p90 = dur
		}
		if 0.99*failCountPerMinute <= failCountPerMinuteBeforeInject && dur < metrics.p99 {
			metrics.p99 = dur
			break
		}
	}
	if err := c.record(&metrics); err != nil {
		return false, err
	}
	return true, nil
}

func (c bankServiceQualityChecker) record(rto *rtoMetrics) error {
	os.MkdirAll(path.Dir(""), 0755)
	f, err := os.OpenFile(c.profFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(rto.nemesis)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(fmt.Sprintf("workload:bank,nemesis:%s,measure_time:%.2f,fail_p80:%.2f,fail_p90:%.2f,fail_p99:%.2f,detail:%s\n",
		rto.nemesis.Name, rto.totalMeasureTime.Seconds(), rto.p80.Seconds(), rto.p90.Seconds(), rto.p99.Seconds(), string(data))); err != nil {
		return err
	}
	return nil
}

func convertToFailSeries(ops []core.Operation) (countWindow, time.Duration) {
	var series []time.Time
	var startTime time.Time
	var endTime time.Time
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
			if startTime.IsZero() {
				startTime = op.Time
			}
			if op.Time.After(endTime) {
				endTime = op.Time
			}
		}
	}
	return countWindow{
		series: series,
	}, endTime.Sub(startTime)
}

// Name returns the name of the verifier.
func (bankServiceQualityChecker) Name() string {
	return "tidb_bank_service_quality_checker"
}

// BankServiceQualityChecker checks the bank history and analyze service quality
func BankServiceQualityChecker(path string) core.Checker {
	return bankServiceQualityChecker{profFile: path}
}
