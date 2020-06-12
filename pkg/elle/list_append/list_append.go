package listappend

import (
	"fmt"

	"github.com/ngaut/log"
	"go.uber.org/zap"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

const unknownPrefixMagicNumber = -114514

// MopValueType type
const initMagicNumber = -114515

// key -> [v0, v1, v2, v3...]
type appendIdx map[string][]core.MopValueType

// key -> v -> Op
type writeIdx map[string]map[core.MopValueType]core.Op

// key -> {v1 -> [Op1, Op2, Op3...], v2 -> [Op4, Op5...]}
type readIdx map[string]map[core.MopValueType][]core.Op

func preprocess(h core.History) (history core.History, appendIdx appendIdx, writeIdx writeIdx, readIdx readIdx) {
	verifyUniqueAppends(h)

	history = core.FilterOkOrInfoHistory(h)
	sortedValueResults := sortedValues(history)
	appendIdx = appendIndex(sortedValueResults)
	writeIdx = writeIndex(history)
	readIdx = readIndex(history)
	return
}

type wwExplainResult struct {
	Typ       core.DependType
	Key       string
	PreValue  core.MopValueType
	Value     core.MopValueType
	AMopIndex int
	BMopIndex int
}

// WWExplainResult creates a wwExplainResult value
func WWExplainResult(key string, preValue, value core.MopValueType, amopIndex, bmopIndex int) wwExplainResult {
	return wwExplainResult{
		Typ:       core.WWDepend,
		Key:       key,
		PreValue:  preValue,
		Value:     value,
		AMopIndex: amopIndex,
		BMopIndex: bmopIndex,
	}
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
			prev := previouslyAppendElement(w.appendIdx, b, bmop)
			if prev == nil {
				continue
			}
			dep := wwMopDep(w.appendIdx, w.writeIdx, b, bmop)
			if dep != nil && *dep == a {
				return WWExplainResult(
					k,
					prev,
					v,
					a.IndexOfMop(core.Append(k, prev.(int))),
					b.IndexOfMop(bmop),
				)
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
func wwGraph(history core.History, _ ...interface{}) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	history, appendIdx, writeIdx, readIdx := preprocess(history)
	g := core.NewDirectedGraph()

	for _, op := range history {
		for _, mop := range *op.Value {
			if !mop.IsAppend() {
				continue
			}
			dep := wwMopDep(appendIdx, writeIdx, op, mop)
			if dep != nil {
				g.Link(core.Vertex{Value: *dep}, core.Vertex{Value: op}, core.WW)
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
	Typ       core.DependType
	Key       string
	Value     core.MopValueType
	AMopIndex int
	BMopIndex int
}

// WRExplainResult creates a wrExplainResult
func WRExplainResult(key string, value core.MopValueType, amopIndex, bmopIndex int) wrExplainResult {
	return wrExplainResult{
		Typ:       core.WRDepend,
		Key:       key,
		Value:     value,
		AMopIndex: amopIndex,
		BMopIndex: bmopIndex,
	}
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
		if !mop.IsRead() {
			continue
		}
		k := mop.GetKey()
		var v []int
		if mop.GetValue() != nil {
			v = mop.GetValue().([]int)
		}
		writer := wrMopDep(w.writeIdx, b, mop)
		if writer != nil && *writer == a {
			return WRExplainResult(
				k,
				v[len(v)-1],
				a.IndexOfMop(core.Append(k, v[len(v)-1])),
				b.IndexOfMop(mop),
			)
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
func wrGraph(history core.History, _ ...interface{}) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	history, appendIdx, writeIdx, readIdx := preprocess(history)
	g := core.NewDirectedGraph()
	for _, op := range history {
		for _, mop := range *op.Value {
			if !mop.IsRead() {
				continue
			}
			dep := wrMopDep(writeIdx, op, mop)
			if dep != nil {
				g.Link(core.Vertex{Value: *dep}, core.Vertex{Value: op}, core.WR)
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
	Typ       core.DependType
	Key       string
	PreValue  core.MopValueType
	Value     core.MopValueType
	AMopIndex int
	BMopIndex int
}

// RWExplainResult creates a rwExplainResult
func RWExplainResult(key string, preValue, value core.MopValueType, amopIndex, bmopIndex int) rwExplainResult {
	return rwExplainResult{
		Typ:       core.RWDepend,
		Key:       key,
		PreValue:  preValue,
		Value:     value,
		AMopIndex: amopIndex,
		BMopIndex: bmopIndex,
	}
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
			prev := previouslyAppendElement(r.appendIdx, b, mop)
			ai := -1
			{
				for i, aMop := range *a.Value {
					if aMop.IsRead() && aMop.GetKey() == k {
						// we should let it panic if vs is not []int
						var vs []int
						if aMop.GetValue() != nil {
							vs = aMop.GetValue().([]int)
						}
						if prev == initMagicNumber && len(vs) == 0 {
							ai = i
							break
						}
						if prev != initMagicNumber && len(vs) != 0 && vs[len(vs)-1] == prev {
							ai = i
							break
						}
					}
				}
			}
			return RWExplainResult(
				k,
				prev,
				v,
				ai,
				i,
			)
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
	if prev == initMagicNumber {
		return fmt.Sprintf("%s observed the initial (nil) state of %s, which %s created by appending %v",
			a, key, b, value,
		)
	}
	return fmt.Sprintf("%s did not observe %s's append of %v to %s",
		a, b, value, key,
	)
}

// rwGraph analyzes read-write anti-dependencies
func rwGraph(history core.History, _ ...interface{}) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
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
func graph(history core.History, _ ...interface{}) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	a, b, c := core.Combine(wwGraph, wrGraph, rwGraph)(history)
	return a, b, c
}

// GCaseTp type aliases []core.Anomaly
type GCaseTp []core.Anomaly

// G1Conflict records a G1 conflict
type G1Conflict struct {
	Op      core.Op
	Mop     core.Mop
	Writer  core.Op
	Element core.MopValueType
}

// IAnomaly ...
func (g G1Conflict) IAnomaly() {}

// String ...
func (g G1Conflict) String() string {
	return fmt.Sprintf("(G1Conflict) Op: %s, mop: %s, writer: %s, element: %v", g.Op, g.Mop.String(), g.Writer.String(), g.Element)
}

// g1aCases finds aborted read cases
func g1aCases(history core.History) GCaseTp {
	failed := txn.FailedWrites(history)
	okHistory := core.FilterOkHistory(history)
	iter := txn.OpMops(okHistory)
	var excepted = make([]core.Anomaly, 0)
	for iter.HasNext() {
		op, mop := iter.Next()
		if mop.IsRead() {
			versionMap := failed[mop.GetKey()]
			var v []int
			if mop.GetValue() != nil {
				v = mop.GetValue().([]int)
			}
			for _, e := range v {
				writer := versionMap[e]
				if writer != nil {
					excepted = append(excepted, G1Conflict{
						Op:      op,
						Mop:     mop,
						Writer:  *writer,
						Element: e,
					})
				}
			}
		}
	}
	return excepted
}

// G1b, or intermediate read, is an anomaly where a transaction T2 reads a
//  state for key k that was written by another transaction, T1, that was not
//  T1's final update to k
func g1bCases(history core.History) GCaseTp {
	inter := txn.IntermediateWrites(history)
	okHistory := core.FilterOkHistory(history)
	iter := txn.OpMops(okHistory)
	var excepted = make([]core.Anomaly, 0)
	for iter.HasNext() {
		op, mop := iter.Next()
		if mop.IsRead() {
			versionMap := inter[mop.GetKey()]
			var v []int
			if mop.GetValue() != nil {
				v = mop.GetValue().([]int)
			}
			if len(v) == 0 {
				continue
			}
			e := v[len(v)-1]
			writer := versionMap[e]
			if writer != nil && *writer != op {
				excepted = append(excepted, G1Conflict{
					Op:      op,
					Mop:     mop,
					Writer:  *writer,
					Element: e,
				})
			}
		}
	}
	return excepted
}

// InternalConflict records a internal conflict
type InternalConflict struct {
	Op       core.Op
	Mop      core.Mop
	Expected core.MopValueType
}

// IAnomaly ...
func (i InternalConflict) IAnomaly() {}

// String ...
func (i InternalConflict) String() string {
	return fmt.Sprintf("(InternalConflict) Op: %s, mop: %s, expected: %s", i.Op, i.Mop.String(), i.Expected)
}

// Note: Please review this function carefully.
// Given an op, returns a Anomaly describing internal consistency violations, or nil otherwise
func opInternalCase(op core.Op) core.Anomaly {
	// key -> valueList
	dataMap := map[string][]int{}
	for _, mop := range *op.Value {
		if mop.IsRead() {
			var records = make([]int, 0)
			if mop.GetValue() != nil {
				records = mop.GetValue().([]int)
			}
			previousData, e := dataMap[mop.GetKey()]
			var found = false
			if e {
				// check conflicts
				if len(previousData) != 0 && previousData[0] == core.MopValueType(unknownPrefixMagicNumber) {
					seqDelta := len(records) + 1 - len(previousData)
					if seqDelta < 0 {
						found = true
					} else {
						found = isReadRecordEqual(previousData[1:], records[seqDelta:]) == false
					}
				} else {
					found = isReadRecordEqual(previousData, records) == false
				}
			}
			if found {
				return InternalConflict{
					Op:       op,
					Mop:      mop,
					Expected: previousData,
				}
			}
			dataMap[mop.GetKey()] = records[:]
		}
		if mop.IsAppend() {
			_, e := dataMap[mop.GetKey()]
			if !e {
				dataMap[mop.GetKey()] = append(dataMap[mop.GetKey()], unknownPrefixMagicNumber)
			}
			dataMap[mop.GetKey()] = append(dataMap[mop.GetKey()], mop.GetValue().(int))
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
	okHistory := core.FilterOkHistory(history)
	for _, op := range okHistory {
		res := opInternalCase(op)
		if res != nil {
			tp = append(tp, res)
		}
	}
	return tp
}

func makeStateTuple(s1, s2 core.OpType) string {
	return fmt.Sprintf("%s__%s", s1, s2)
}

// DirtyUpdateConflict records a dirty update conflict
type DirtyUpdateConflict struct {
	Key      string
	Values   []core.MopValueType
	Op1, Op2 core.Op
}

// IAnomaly ...
func (d DirtyUpdateConflict) IAnomaly() {}

func (d DirtyUpdateConflict) String() string {
	return fmt.Sprintf("(DirtyUpdateConflict) Key is %s, op1 is %s, op2 is %s, values %v", d.Key, d.Op1, d.Op2, d.Values)
}

func dirtyUpdateCases(appendIndexResult map[string][]core.MopValueType, history core.History) GCaseTp {
	wi := writeIndex(history)
	var cases []core.Anomaly

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
				cases = append(cases, DirtyUpdateConflict{
					Key:    key,
					Values: currentMayAnomalyValues,
					Op1:    currentOp,
					Op2:    writer,
				})
				switchCurrentState()
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
	historyOKOrInfo := core.FilterOkOrInfoHistory(history)
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
		anomalies["incompatible-order"] = incmpOrder
	}
	if len(internal) != 0 {
		anomalies["internal"] = internal
	}
	if len(dirtyUpdate) != 0 {
		anomalies["dirty-update"] = dirtyUpdate
	}
	if len(g1a) != 0 {
		anomalies["G1a"] = g1a
	}
	if len(g1b) != 0 {
		anomalies["G1b"] = g1b
	}
	return txn.ResultMap(opts, anomalies)
}
