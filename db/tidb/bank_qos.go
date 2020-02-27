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
	name             string
	nemesis          *core.NemesisGeneratorRecord
	totalMeasureTime time.Duration
	baseline         float64
	p80              time.Duration
	p90              time.Duration
	p99              time.Duration
}

func (m *rtoMetrics) Record(filePath string) error {
	os.MkdirAll(path.Dir(""), 0755)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(m.nemesis)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(fmt.Sprintf("workload:%s,nemesis:%s,measure_time:%.2f,baseline:%.2f,tpm_p80:%.2f,tpm_p90:%.2f,tpm_p99:%.2f,detail:%s\n",
		m.name,
		m.nemesis.Name,
		m.totalMeasureTime.Seconds(),
		m.baseline,
		m.p80.Seconds(), m.p90.Seconds(), m.p99.Seconds(), string(data))); err != nil {
		return err
	}
	return nil
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

type bankQoSChecker struct {
	summaryFile string
}

// bankQoSChecker checks the bank history and measure failure requests count before and after nemesis was injected.
// When the avg failure count of one continuous minute is below 0.8 * the failure count before nemesis, we say the the system
// recover to p80 line from that time point. And p90 or p99 are similar.
func (c bankQoSChecker) Check(_ core.Model, ops []core.Operation) (bool, error) {
	var (
		nemesisRecord core.NemesisGeneratorRecord
		injectionTime time.Time
		window        countWindow
		duration      = time.Minute
	)

	metrics := rtoMetrics{name: "bank", nemesis: &nemesisRecord}
	metrics.nemesis, metrics.baseline, injectionTime, window, metrics.totalMeasureTime = convertToFailSeries(ops)

	if len(window.series) == 0 {
		metrics.p80, metrics.p90, metrics.p99 = time.Duration(0), time.Duration(0), time.Duration(0)
	} else {
		windowDuration := window.series[len(window.series)-1].Sub(injectionTime)
		metrics.p80, metrics.p90, metrics.p99 = windowDuration, windowDuration, windowDuration
	}
	for window.Next() {
		leftBound, failCount := window.Count(duration)
		dur := leftBound.Sub(injectionTime)
		failCountPerMinute := float64(failCount) / duration.Minutes()

		if 0.8*failCountPerMinute <= metrics.baseline && dur < metrics.p80 {
			metrics.p80 = dur
		}
		if 0.9*failCountPerMinute <= metrics.baseline && dur < metrics.p90 {
			metrics.p90 = dur
		}
		if 0.99*failCountPerMinute <= metrics.baseline && dur < metrics.p99 {
			metrics.p99 = dur
			break
		}
	}
	if err := metrics.Record(c.summaryFile); err != nil {
		return false, err
	}
	return true, nil
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
func (bankQoSChecker) Name() string {
	return "tidb_bank_QoS_checker"
}

// BankQoSChecker checks the bank history and analyze QoS
func BankQoSChecker(path string) core.Checker {
	return bankQoSChecker{summaryFile: path}
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
