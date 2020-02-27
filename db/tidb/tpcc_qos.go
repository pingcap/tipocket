package tidb

import (
	"log"
	"sort"
	"time"

	"github.com/pingcap/tipocket/pkg/core"
)

type tpccQoSChecker struct {
	warmUpDuration time.Duration
	summaryFile    string
}

func TPCCQosChecker(warmUpDuration time.Duration, path string) core.Checker {
	return &tpccQoSChecker{warmUpDuration: warmUpDuration, summaryFile: path}
}

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
		if float64(tpm) >= metrics.baseline*0.9 && dur < metrics.p80 {
			metrics.p90 = dur
		}
		if float64(tpm) >= metrics.baseline*0.99 && dur < metrics.p80 {
			metrics.p99 = dur
			break
		}
	}
	if err := metrics.Record(t.summaryFile); err != nil {
		return false, err
	}
	return true, nil
}

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
			if response.Error == "" {
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
