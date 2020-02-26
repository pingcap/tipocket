package tidb

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"time"

	"github.com/pingcap/tipocket/pkg/core"
)

type tpccQoSChecker struct {
	summaryFile string
}

func TPCCQosChecker(path string) core.Checker {
	return &tpccQoSChecker{summaryFile: path}
}

func (t *tpccQoSChecker) Check(_ core.Model, ops []core.Operation) (bool, error) {
	var (
		nemesisRecord core.NemesisGeneratorRecord
		baselineTPM   float64
		injectionTime time.Time
		window        countWindow
	)

	metrics := rtoMetrics{nemesis: &nemesisRecord}
	metrics.nemesis, baselineTPM, injectionTime, window, metrics.totalMeasureTime = convertToTPMSeries(ops)

	if len(window.series) == 0 {
		metrics.p80, metrics.p90, metrics.p99 = time.Duration(0), time.Duration(0), time.Duration(0)
	} else {
		windowDuration := window.series[len(window.series)-1].Sub(injectionTime)
		metrics.p80, metrics.p90, metrics.p99 = windowDuration, windowDuration, windowDuration
	}
	for window.Next() {
		leftBound, tpm := window.Count(time.Minute)
		dur := leftBound.Sub(injectionTime)

		if float64(tpm) >= baselineTPM*0.8 && dur < metrics.p80 {
			metrics.p80 = dur
		}
		if float64(tpm) >= baselineTPM*0.9 && dur < metrics.p80 {
			metrics.p90 = dur
		}
		if float64(tpm) >= baselineTPM*0.99 && dur < metrics.p80 {
			metrics.p99 = dur
			break
		}
	}
	if err := t.record(&metrics); err != nil {
		return false, err
	}
	return true, nil
}

func (t *tpccQoSChecker) record(rto *rtoMetrics) error {
	os.MkdirAll(path.Dir(""), 0755)
	f, err := os.OpenFile(t.summaryFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(rto.nemesis)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(fmt.Sprintf("workload:tpcc,nemesis:%s,measure_time:%.2f,tpm_p80:%.2f,tpm_p90:%.2f,tpm_p99:%.2f,detail:%s\n",
		rto.nemesis.Name, rto.totalMeasureTime.Seconds(), rto.p80.Seconds(), rto.p90.Seconds(), rto.p99.Seconds(), string(data))); err != nil {
		return err
	}
	return nil
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
			// TODO: check Error of response?
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

	sort.Sort(timeSeries(series))
	window = countWindow{series: series}
	measureDuration = endTime.Sub(startTime)
	baselineTPM = float64(baselineTransaction) / injectionTime.Sub(startTime).Minutes()

	return
}

func (t *tpccQoSChecker) Name() string {
	return "tpcc_QoS_checker"
}
