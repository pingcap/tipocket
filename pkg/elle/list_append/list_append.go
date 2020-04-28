package list_append

import (
	"fmt"

	"github.com/ngaut/log"
	"go.uber.org/zap"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

// key -> [v0, v1, v2, v3...]
type appendIdx map[string][]core.MopValueType

// key -> v -> Op
type writeIdx map[string]map[core.MopValueType]core.Op

// key -> [Op1, Op2, Op3...]
type readIdx map[string][]core.Op

func preprocess(h core.History) (history core.History, appendIdx appendIdx, writeIdx writeIdx, readIdx readIdx) {
	panic("impl me")
}

type wwExplainResult struct {
	Key       string
	PreValue  core.MopValueType
	Value     core.MopValueType
	AMopIndex int
	BMopIndex int
}

func (w wwExplainResult) Type() core.DependType {
	return core.WWDepend
}

// wwExplainer explains write-write dependencies
type wwExplainer struct {
	appendIdx
	writeIdx
	readIdx
}

func (w *wwExplainer) ExplainPairData(a, b core.PathType) core.ExplainResult {
	for _, bmop := range *b.Value {
		k := bmop.GetKey()
		v := bmop.GetValue()
		if bmop.IsAppend() {
			prev, isInit := previouslyAppendElement(w.appendIdx, b, bmop)
			// if bmop is the first mop of append key k
			// or some logic error
			if prev == nil || isInit {
				continue
			}
			dep := wwMopDep(w.appendIdx, w.writeIdx, b, bmop)
			// TODO(mahjonp): should panic if dep == nil?
			if dep != nil && *dep == a {
				return wwExplainResult{
					Key:      k,
					PreValue: prev,
					Value:    v,
					AMopIndex: a.IndexOfMop(core.Append{
						Key:   k,
						Value: v,
					}),
					BMopIndex: b.IndexOfMop(bmop),
				}
			}
		}
	}
	return nil
}

func (w *wwExplainer) RenderExplanation(result core.ExplainResult, a, b string) string {
	if result.Type() != core.WWDepend {
		log.Fatalf("result type is not %s, type error", core.WWDepend)
	}
	er := result.(wwExplainResult)
	return fmt.Sprintf("%s appended %v after %s appended %v to %s",
		b, er.Value, a, er.PreValue, er.Key,
	)
}

// wwGraph analyzes write-write dependencies
func wwGraph(history core.History) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	history, appendIdx, writeIdx, readIdx := preprocess(history)
	g := core.NewDirectedGraph()

	for _, op := range history {
		for _, mop := range *op.Value {
			if !mop.IsAppend() {
				continue
			}
			dep := wwMopDep(appendIdx, writeIdx, op, mop)
			if dep != nil {
				g.Link(core.Vertex{Value: dep}, core.Vertex{Value: op}, core.WW)
			}
		}
	}
	return nil, g, &wwExplainer{
		appendIdx: appendIdx,
		writeIdx:  writeIdx,
		readIdx:   readIdx,
	}
}

type wrExplainResult struct {
	Key       string
	Value     core.MopValueType
	AMopIndex int
	BMopIndex int
}

func (w wrExplainResult) Type() core.DependType {
	return core.WRDepend
}

// wrExplainer explains write-read dependencies
type wrExplainer struct {
	appendIdx
	writeIdx
	readIdx
}

func (w *wrExplainer) ExplainPairData(a, b core.PathType) core.ExplainResult {
	for _, mop := range *b.Value {
		k := mop.GetKey()
		v := mop.GetValue().([]core.Mop)
		if !mop.IsRead() {
			continue
		}
		writer := wrMopDep(w.writeIdx, b, mop)
		if writer != nil && *writer == a {
			return wrExplainResult{
				Key:   k,
				Value: v[0],
				AMopIndex: a.IndexOfMop(core.Append{
					Key:   k,
					Value: v[0],
				}),
				BMopIndex: b.IndexOfMop(mop),
			}
		}
	}
	return nil
}

func (w *wrExplainer) RenderExplanation(result core.ExplainResult, a, b string) string {
	if result.Type() != core.WRDepend {
		log.Fatalf("result type is not %s, type error", core.WRDepend)
	}
	er := result.(wrExplainResult)
	return fmt.Sprintf("%s observed %s's append of %v to key %s",
		b, a, er.Value, er.Key,
	)
}

// wrGraph analyzes write-read dependencies
func wrGraph(history core.History) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	history, appendIdx, writeIdx, readIdx := preprocess(history)
	g := core.NewDirectedGraph()
	for _, op := range history {
		for _, mop := range *op.Value {
			if !mop.IsRead() {
				continue
			}
			dep := wrMopDep(writeIdx, op, mop)
			if dep != nil {
				g.Link(core.Vertex{Value: dep}, core.Vertex{Value: op}, core.WR)
			}
		}
	}
	return nil, g, &wrExplainer{
		appendIdx: appendIdx,
		writeIdx:  writeIdx,
		readIdx:   readIdx,
	}
}

type rwExplainResult struct {
	Key       string
	PreValue  core.MopValueType
	Value     core.MopValueType
	AMopIndex int
	BMopIndex int
}

func (w rwExplainResult) Type() core.DependType {
	return core.RWDepend
}

// rwExplainer explains read-write anti-dependencies
type rwExplainer struct {
	appendIdx
	writeIdx
	readIdx
}

func (r *rwExplainer) ExplainPairData(a, b core.PathType) core.ExplainResult {
	for i, mop := range *b.Value {
		k := mop.GetKey()
		v := mop.GetValue()
		if !mop.IsAppend() {
			continue
		}
		readers := rwMopDeps(r.appendIdx, r.writeIdx, r.readIdx, b, mop)
		if _, ok := readers[a]; ok {
			prev, _ := previouslyAppendElement(r.appendIdx, b, mop)
			ai := -1
			for i, aMop := range *a.Value {
				if aMop.IsRead() && aMop.GetKey() == k {
					vs := aMop.GetValue().([]int)
					if prev == nil && len(vs) == 0 {
						ai = i
						break
					}
					if prev != nil && len(vs) != 0 && vs[len(vs)-1] == prev {
						ai = i
						break
					}
				}
			}
			return rwExplainResult{
				Key:       k,
				PreValue:  prev,
				Value:     v,
				AMopIndex: ai,
				BMopIndex: i,
			}
		}
	}
	return nil
}

func (r *rwExplainer) RenderExplanation(result core.ExplainResult, a, b string) string {
	if result.Type() != core.RWDepend {
		log.Fatalf("result type is not %s, type error", core.RWDepend)
	}
	er := result.(rwExplainResult)
	key, prev, value := er.Key, er.PreValue, er.Value
	if prev == nil {
		return fmt.Sprintf("%s observed the initial (nil) state of %s, which %s created by appending %v",
			a, key, b, value,
		)
	}
	return fmt.Sprintf("%s did not observe %s's append of %v to %s",
		a, b, value, key,
	)
}

// rwGraph analyzes read-write anti-dependencies
func rwGraph(history core.History) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	history, appendIdx, writeIdx, readIdx := preprocess(history)
	g := core.NewDirectedGraph()

	for _, op := range history {
		for _, mop := range *op.Value {
			if mop.IsAppend() {
				deps := rwMopDeps(appendIdx, writeIdx, readIdx, op, mop)
				var vdeps []core.Vertex
				for dep := range deps {
					vdeps = append(vdeps, core.Vertex{Value: dep})
				}
				g.LinkAllTo(vdeps, core.Vertex{Value: op}, core.RW)
			}
		}
	}

	return nil, g, &rwExplainer{
		appendIdx: appendIdx,
		writeIdx:  writeIdx,
		readIdx:   readIdx,
	}
}

// graph combines wwGraph, wrGraph and rwGraph
func graph(history core.History) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	return core.Combine(wwGraph, wrGraph, rwGraph)(history)
}

type GCase struct {
	Operation core.Op
	Mop       core.Mop

	Writer  core.Op
	Element core.MopValueType

	Expected interface{}
}

type GCaseTp []GCase

type G1aConflict struct {
	Op      core.Op
	Mop     core.Mop
	Writer  core.Op
	Element core.MopValueType
}

func g1aCases(history core.History) GCaseTp {
	failed := txn.FailedWrites(history)
	okHistory := filterOkHistory(history)
	iter := txn.OpMops(okHistory)
	var excepted []GCase
	for iter.HasNext() {
		op, mop := iter.Next()
		if mop.IsRead() {
			versionMap := failed[mop.GetKey()]
			v := mop.GetValue().(int)
			writer := versionMap[v]
			excepted = append(excepted, GCase{
				Operation: *op,
				Mop:       mop,
				Writer:    *writer,
				Element:   mop.GetValue(),
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
	okHistory := filterOkHistory(history)
	iter := txn.OpMops(okHistory)
	var excepted []GCase
	for iter.HasNext() {
		op, mop := iter.Next()
		if mop.IsRead() {
			versionMap := inter[mop.GetKey()]
			v := mop.GetValue().(int)
			writer := versionMap[v]
			excepted = append(excepted, GCase{
				Operation: *op,
				Mop:       mop,
				Writer:    *writer,
				Element:   mop.GetValue(),
			})
		}
	}
	return excepted
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
						Mop:       v,
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
						Mop:       v,
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
	okHistory := filterOkHistory(history)
	for _, op := range okHistory {
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
	history = preProcessHistory(history)
	g1a := g1aCases(history)
	g1b := g1bCases(history)
	internal := internalCases(history)
	dirtyUpdate := dirtyUpdateCases(appendIndex(sortedValues(history)), history)
	historyOKOrInfo := filterOkOrInfoHistory(history)
	dups := duplicates(historyOKOrInfo)
	sortedValues := sortedValues(historyOKOrInfo)
	incmpOrder := incompatibleOrders(sortedValues)
	var analyzer core.Analyzer = graph
	additionalGraphs := txn.AdditionalGraphs(opts)
	if len(additionalGraphs) != 0 {
		analyzer = core.Combine(append([]core.Analyzer{analyzer}, additionalGraphs...)...)
	}

	checkResult := txn.Cycles(analyzer, history)
	anomalies := checkResult.Anomalies
	if len(dups) != 0 {
		anomalies["duplicate-elements"] = dups
	}
	if len(incmpOrder) != 0 {
		anomalies["incompatible-order"] = dups
	}
	if len(internal) != 0 {
		anomalies["internal"] = dups
	}
	if len(dirtyUpdate) != 0 {
		anomalies["dirty-update"] = dups
	}
	if len(g1a) != 0 {
		anomalies["G1a"] = dups
	}
	if len(g1b) != 0 {
		anomalies["G1b"] = dups
	}
	return txn.ResultMap(opts, anomalies)
}
