package list_append

import (
	"fmt"

	"github.com/pingcap/log"
	"go.uber.org/zap"

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

const InternalMagicNumber = -114514

// Note: Please review this function carefully.
func opInternalCases(op core.Op) *GCase {
	// key -> valueList
	dataMap := map[string][]core.MopValueType{}
	for _, v := range *op.Value {
		if v.IsRead() {
			previousData, e := dataMap[v.GetKey()]
			readData := v.(core.Read)
			records := readData.Value.([]int)
			if !e {
				for _, rdata := range records {
					dataMap[v.GetKey()] = append(dataMap[v.GetKey()], core.MopValueType(rdata))
				}
				continue
			}
			// check conflicts
			comIndex := 0
			if len(previousData) > 0 && previousData[0] == core.MopValueType(InternalMagicNumber) {
				seqDelta := len(records) + 1 - len(previousData)
				if seqDelta < 0 {
					return &GCase{
						Operation: op,
						Mop:       &v,
						Expected:  previousData,
					}
				} else {
					comIndex = seqDelta
				}
			}
			for i, indexV := range records[comIndex:] {
				if previousData[i+1].(int) != indexV {
					return &GCase{
						Operation: op,
						Mop:       &v,
						Expected:  previousData,
					}
				}
			}
			for _, rdata := range records {
				dataMap[v.GetKey()] = append(dataMap[v.GetKey()], core.MopValueType(rdata))
			}
		} else {
			// Note: current we only support read and append. So if it's not
			//  read, it must be append.
			_, e := dataMap[v.GetKey()]
			if !e {
				dataMap[v.GetKey()] = append(dataMap[v.GetKey()], InternalMagicNumber)
			}
			dataMap[v.GetKey()] = append(dataMap[v.GetKey()], v.GetValue())
		}
	}
	return nil
}

// Given an op, returns a map describing internal consistency violations, or
//  nil otherwise. Our maps are:
// {:op        The operation which went wrong
//  :mop       The micro-operation which went wrong
//  :expected  The state we expected to observe. Either a definite list
//                  like [1 2 3] or a postfix like ['... 3]}"
func internalCases(history core.History) GCaseTp {
	var tp GCaseTp
	for _, op := range history {
		res := opInternalCases(op)
		if res != nil {
			tp = append(tp, *res)
		}
	}
	return tp
}

func makeStateTuple(s1, s2 core.OpType) string {
	return fmt.Sprintf("%s__%s", s1, s2)
}

func dirtyUpdateCases(appendIndexResult map[string][]core.MopValueType, history core.History) GCaseTp {
	wi := writeIndex(history)

	var cases []GCase

	for key, valueHistory := range appendIndexResult {
		var currentMayAnomalyValues []core.MopValueType
		currentOp := core.Op{Type: core.OpTypeOk}
		keyWriter, e := wi[key]
		if !e {
			log.Warn("The key doesn't has any writer, the code may has bug", zap.String("key", key))
			continue
		}
		for _, v := range valueHistory {
			currentMayAnomalyValues = append(currentMayAnomalyValues, v)
			writer, e := keyWriter[v]
			if !e {
				// value doesn't has any writer
				log.Warn("The value doesn't has any writer, the code may has bug", zap.String("key", key))
				continue
			}

			switchCurrentState := func() {
				currentOp = writer
				currentMayAnomalyValues = []core.MopValueType{v}
			}

			// do nothing
			switchWriterState := func() {}

			switch makeStateTuple(currentOp.Type, writer.Type) {
			case makeStateTuple(core.OpTypeOk, core.OpTypeOk):
				switchCurrentState()
			case makeStateTuple(core.OpTypeInfo, core.OpTypeOk):
				switchCurrentState()
			case makeStateTuple(core.OpTypeFail, core.OpTypeOk):
				// Find a bug!
				// TODO: adding type for it
				switchCurrentState()
				cases = append(cases, GCase{})
			case makeStateTuple(core.OpTypeOk, core.OpTypeInfo):
				switchWriterState()
			case makeStateTuple(core.OpTypeInfo, core.OpTypeInfo):
				switchWriterState()
			case makeStateTuple(core.OpTypeFail, core.OpTypeInfo):
				switchWriterState()
			case makeStateTuple(core.OpTypeOk, core.OpTypeFail):
				switchCurrentState()
			case makeStateTuple(core.OpTypeInfo, core.OpTypeFail):
				switchCurrentState()
			case makeStateTuple(core.OpTypeFail, core.OpTypeFail):
				switchWriterState()
			}

			//if currentOp.Type == core.OpTypeFail {
			//	if writer.Type == core.OpTypeInfo || writer.Type == core.OpTypeFail {
			//		// why writer will have :info, I don't know :(
			//		continue
			//	}
			//	if writer.Type == core.OpTypeOk {
			//		// Yes, we found a bug.
			//		// TODO: change the case type.
			//		cases = append(cases, GCase{
			//
			//		})
			//		currentMayAnomalyValues = []core.MopValueType{v}
			//		currentOp = writer
			//	}
			//} else if currentOp.Type == core.OpTypeInfo {
			//	if writer.Type == core.OpTypeInfo {
			//		continue
			//	} else {
			//		currentOp = writer
			//
			//	}
			//}
		}
	}

	return cases
}

// Check checks append and read history for list_append
func Check(opts txn.Opts, history core.History) txn.CheckResult {
	panic("implement me")
}
