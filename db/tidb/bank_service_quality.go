package tidb

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
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
	var (
		count     = 0
		pos       = f.pos
		startTime = f.series[pos]
	)

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
	var (
		nemesisRecord                  core.NemesisGeneratorRecord
		failCountPerMinuteBeforeInject float64
		injectionTime                  time.Time
		window                         countWindow
		duration                       = time.Minute
	)

	metrics := rtoMetrics{nemesis: &nemesisRecord}
	metrics.nemesis, failCountPerMinuteBeforeInject, injectionTime, window, metrics.totalMeasureTime = convertToFailSeries(ops)

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

func convertToFailSeries(ops []core.Operation) (nemesisRecord *core.NemesisGeneratorRecord,
	baselineFailCount float64,
	injectionTime time.Time,
	window countWindow,
	measureDuration time.Duration) {

	var (
		series    []time.Time
		startTime time.Time
		endTime   time.Time
		opMap     = make(map[int64]*core.Operation)
	)

	for idx := range ops {
		op := ops[idx]
		if op.Action == core.InvokeOperation {
			opMap[op.Proc] = &op
		} else if op.Action == core.ReturnOperation {
			response := op.Data.(bankResponse)
			if response.Unknown {
				invokeOps := opMap[op.Proc]
				if injectionTime.IsZero() {
					log.Fatalf("illegal ops, the invokeOperation item must before any unknown return ops")
				}
				if invokeOps.Time.Before(injectionTime) {
					baselineFailCount++
				} else {
					series = append(series, invokeOps.Time)
				}
			} else {
				delete(opMap, op.Proc)
			}
		} else if op.Action == core.InvokeNemesis {
			injectionTime = op.Time
			record := op.Data.(core.NemesisGeneratorRecord)
			nemesisRecord = &record
		}
		if startTime.IsZero() {
			startTime = op.Time
		}
		if op.Time.After(endTime) {
			endTime = op.Time
		}
	}

	sort.Sort(timeSeries(series))
	window = countWindow{series: series}
	measureDuration = endTime.Sub(startTime)

	return
}

// Name returns the name of the verifier.
func (bankServiceQualityChecker) Name() string {
	return "tidb_bank_service_quality_checker"
}

// BankServiceQualityChecker checks the bank history and analyze service quality
func BankServiceQualityChecker(path string) core.Checker {
	return bankServiceQualityChecker{profFile: path}
}

type timeSeries []time.Time

func (t timeSeries) Len() int {
	return len(t)
}

func (t timeSeries) Less(i, j int) bool {
	return t[i].Before(t[j])
}

func (t timeSeries) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
