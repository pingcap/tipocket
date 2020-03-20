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

type tpccQoSChecker struct {
	warmUpDuration time.Duration
	summaryFile    string
}

// TPCCQosChecker creates a tpccQoSChecker instance
func TPCCQosChecker(warmUpDuration time.Duration, path string) core.Checker {
	return &tpccQoSChecker{warmUpDuration: warmUpDuration, summaryFile: path}
}

// Check checks QoS of tpcc workload
func (t *tpccQoSChecker) Check(_ core.Model, ops []core.Operation) (bool, error) {
	var (
		nemesisRecord core.NemesisGeneratorRecord
		injectionTime time.Time
		window        countWindow
	)
	ops = dropWarmUpOps(ops, t.warmUpDuration)
	metrics := rtoMetrics{name: "tpcc", nemesis: &nemesisRecord}
	metrics.nemesis, metrics.baseline, injectionTime, window, metrics.totalMeasureTime = convertToTPMSeries(ops)

	if len(window.series) == 0 {
		metrics.p80, metrics.p90, metrics.p99 = time.Duration(0), time.Duration(0), time.Duration(0)
	} else {
		windowDuration := window.series[len(window.series)-1].Sub(injectionTime)
		metrics.p80, metrics.p90, metrics.p99 = windowDuration, windowDuration, windowDuration
	}
	for window.Next() {
		leftBound, tpm := window.Count(time.Minute)
		dur := leftBound.Sub(injectionTime)
		if float64(tpm) >= metrics.baseline*0.8 && dur < metrics.p80 {
			metrics.p80 = dur
		}
		if float64(tpm) >= metrics.baseline*0.9 && dur < metrics.p90 {
			metrics.p90 = dur
		}
		if float64(tpm) >= metrics.baseline*0.99 && dur < metrics.p99 {
			metrics.p99 = dur
			break
		}
	}
	if err := metrics.Record(t.summaryFile); err != nil {
		return false, err
	}
	return true, nil
}

// Name returns the checker's name
func (t *tpccQoSChecker) Name() string {
	return "tpcc_QoS_checker"
}

func dropWarmUpOps(ops []core.Operation, warmUpDuration time.Duration) []core.Operation {
	var filterOps []core.Operation
	if len(ops) < 2 {
		return ops
	}
	warmUpEnd := ops[0].Time.Add(warmUpDuration)
	for _, op := range ops {
		switch op.Action {
		case core.ReturnOperation:
			if op.Time.After(warmUpEnd) {
				filterOps = append(filterOps, op)
			}
		case core.InvokeNemesis, core.RecoverNemesis:
			filterOps = append(filterOps, op)
		}
	}
	return filterOps
}

func convertToTPMSeries(ops []core.Operation) (nemesisRecord *core.NemesisGeneratorRecord,
	baselineTPM float64,
	injectionTime time.Time,
	window countWindow,
	measureDuration time.Duration) {

	var (
		baselineTransaction = 0
		series              []time.Time
		startTime           time.Time
		endTime             time.Time
	)

	for idx := range ops {
		op := ops[idx]
		if op.Action == core.ReturnOperation {
			response := op.Data.(tpccResponse)
			if response.Error != "" {
				// skip Error response
				continue
			}
			if injectionTime.IsZero() {
				baselineTransaction++
			} else {
				series = append(series, op.Time)
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

	if injectionTime.IsZero() {
		log.Fatalf("illegal ops to measure QoS, lacking the InvokeNemesis record")
	}

	sort.Sort(timeSeries(series))
	window = countWindow{series: series}
	measureDuration = endTime.Sub(startTime)
	baselineTPM = float64(baselineTransaction) / injectionTime.Sub(startTime).Minutes()

	return
}
