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

type GCase struct {
	Operation core.Op
	Mop       *core.Mop

	Writer  core.Op
	Element core.MopValueType

	Expected interface{}
}

type GCaseTp []GCase

type G1aConflict struct {
	Op      core.Op
	Mop     *core.Mop
	Writer  core.Op
	Element core.MopValueType
}

func g1aCases(history core.History) GCaseTp {
	failed := txn.FailedWrites(history)
	okHistory := okHistoryFilter(history)
	iter := txn.OpMops(okHistory)
	var excepted []GCase
	for iter.HasNext() {
		op, mop := iter.Next()
		if (*mop).IsRead() {
			versionMap := failed[(*mop).GetKey()]
			v := (*mop).GetValue().(int)
			writer := versionMap[v]
			excepted = append(excepted, GCase{
				Operation: *op,
				Mop:       mop,
				Writer:    *writer,
				Element:   (*mop).GetValue(),
			})
		}
	}
	return excepted
}

// G1b, or intermediate read, is an anomaly where a transaction T2 reads a
//  state for key k that was written by another transaction, T1, that was not
//  T1's final update to k
func g1bCases(history core.History) GCaseTp {
	inter := txn.IntermediateWrites(history)
	okHistory := okHistoryFilter(history)
	iter := txn.OpMops(okHistory)
	var excepted []GCase
	for iter.HasNext() {
		op, mop := iter.Next()
		if (*mop).IsRead() {
			versionMap := inter[(*mop).GetKey()]
			v := (*mop).GetValue().(int)
			writer := versionMap[v]
			excepted = append(excepted, GCase{
				Operation: *op,
				Mop:       mop,
				Writer:    *writer,
				Element:   (*mop).GetValue(),
			})
		}
	}
	return excepted
}

func okHistoryFilter(history core.History) core.History {
	var okHistory []core.Op
	for _, v := range history {
		okHistory = append(okHistory, v)
	}
	return okHistory
}

func opInternalCases(op core.Op) bool {

	for _, v := range *op.Value {
		if v.IsRead() {

		} else {
			// Note: current we only support read and append. So if it's not
			//  read, it must be append.

		}
	}
	panic("implement me")
}

// Given an op, returns a map describing internal consistency violations, or
//  nil otherwise. Our maps are:
// {:op        The operation which went wrong
//  :mop       The micro-operation which went wrong
//  :expected  The state we expected to observe. Either a definite list
//                  like [1 2 3] or a postfix like ['... 3]}"
func internalCases(history core.History) GCaseTp {
	panic("implement me")
}

func dirtyUpdateCases(history core.History) GCaseTp {
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
