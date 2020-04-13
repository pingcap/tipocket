package list_append

import (
	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

type appendIdx struct{}
type writeIdx struct{}
type readIdx struct{}

func preprocess(h core.History) (history core.History, appendIdx appendIdx, writeIdx writeIdx, readIdx readIdx) {
	panic("impl me")
}

// wwExplainer explains write-write dependencies
type wwExplainer struct {
	appendIdx
	writeIdx
	readIdx
}

func (W *wwExplainer) ExplainPairData() interface{} {
	panic("implement me")
}

func (W *wwExplainer) RenderExplanation() string {
	panic("implement me")
}

// wwGraph analyzes write-write dependencies
func wwGraph(history core.History) (core.Anomalies, core.DirectedGraph, core.DataExplainer) {
	panic("implement me")
}

// wrExplainer explains write-read dependencies
type wrExplainer struct {
	appendIdx
	writeIdx
	readIdx
}

func (w *wrExplainer) ExplainPairData() interface{} {
	panic("implement me")
}

func (w *wrExplainer) RenderExplanation() string {
	panic("implement me")
}

// wrGraph analyzes write-read dependencies
func wrGraph(history core.History) (core.Anomalies, core.DirectedGraph, core.DataExplainer) {
	panic("implement me")
}

// rwExplainer explains read-write anti-dependencies
type rwExplainer struct {
	appendIdx
	writeIdx
	readIdx
}

func (r *rwExplainer) ExplainPairData() interface{} {
	panic("implement me")
}

func (r *rwExplainer) RenderExplanation() string {
	panic("implement me")
}

// rwGraph analyzes read-write anti-dependencies
func rwGraph(history core.History) (core.Anomalies, core.DirectedGraph, core.DataExplainer) {
	panic("implement me")
}

// graph combines wwGraph, wrGraph and rwGraph
func graph(history core.History) (core.Anomalies, core.DirectedGraph, core.DataExplainer) {
	panic("implement me")
}

func g1aCases(history core.History) interface{} {
	panic("implement me")
}

func g1bCases(history core.History) interface{} {
	panic("implement me")
}

func internalCases(history core.History) interface{} {
	panic("implement me")
}

func dirtyUpdateCases(history core.History) interface{} {
	panic("implement me")
}

func duplicates(history core.History) interface{} {
	panic("implement me")
}

func incompatibleOrders(history core.History) interface{} {
	panic("implement me")
}

// Check checks append and read history for list_append
func Check(opts txn.Opts, history core.History) txn.CheckResult {
	panic("implement me")
}
