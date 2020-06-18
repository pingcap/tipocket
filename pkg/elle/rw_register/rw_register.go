package rwregister

import (
	"fmt"
	"log"
	"sort"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

// GraphOption ...
type GraphOption struct {
	LinearizableKeys bool //Uses realtime order
	SequentialKeys   bool // Uses process order
	WfrKeys          bool // Assumes writes follow reads in a txn
}

// GCaseTp type aliases []core.Anomaly
type GCaseTp []core.Anomaly

// InternalConflict records a internal conflict
type InternalConflict struct {
	Op       core.Op
	Mop      core.Mop
	Expected core.Mop
}

// IAnomaly ...
func (i InternalConflict) IAnomaly() {}

// String ...
func (i InternalConflict) String() string {
	return fmt.Sprintf("(InternalConflict) Op: %s, mop: %s, expected: %s", i.Op, i.Mop.String(), i.Expected)
}

func internalOp(op core.Op) core.Anomaly {
	dataMap := make(map[string]Int)

	for _, mop := range *op.Value {
		if mop.IsWrite() {
			k, v := mop.GetKey(), mop.GetValue()
			vint := v.(Int)
			if vint.IsNil {
				panic("write value should not be nil")
			}
			dataMap[k] = vint
		}
		if mop.IsRead() {
			k, v := mop.GetKey(), mop.GetValue()
			vint := v.(Int)
			if vint.IsNil {
				// skip failed read
				continue
			}
			if prev, ok := dataMap[k]; !ok {
				dataMap[k] = vint
			} else {
				if !prev.Eq(vint) {
					expected := mop.Copy()
					expected.M["key"] = k
					expected.M["value"] = prev
					return InternalConflict{
						Op:       op,
						Mop:      mop,
						Expected: expected,
					}
				}
			}
		}
	}
	return nil
}

func internal(history core.History) GCaseTp {
	var tp GCaseTp
	okHistory := core.FilterOkHistory(history)
	for _, op := range okHistory {
		res := internalOp(op)
		if res != nil {
			tp = append(tp, res)
		}
	}
	return tp
}

// G1Conflict records a G1 conflict
type G1Conflict struct {
	Op      core.Op
	Mop     core.Mop
	Writer  core.Op
	Element string
}

// IAnomaly ...
func (g G1Conflict) IAnomaly() {}

// String ...
func (g G1Conflict) String() string {
	return fmt.Sprintf("(G1Conflict) Op: %s, mop: %s, writer: %s, element: %s", g.Op, g.Mop.String(), g.Writer.String(), g.Element)
}

func g1aCases(history core.History) GCaseTp {
	failedMap := make(map[core.KVEntity]core.Op)
	var anomalies []core.Anomaly

	for _, op := range core.FilterFailedHistory(history) {
		for _, mop := range *op.Value {
			if mop.IsWrite() {
				k, v := mop.GetKey(), mop.GetValue()
				vint := v.(Int)
				if vint.IsNil {
					panic("write value should not be nil")
				}
				failedMap[core.KVEntity{K: k, V: vint}] = op
			}
		}
	}

	for _, op := range core.FilterOkHistory(history) {
		for _, mop := range *op.Value {
			if mop.IsRead() {
				k, v := mop.GetKey(), mop.GetValue()
				vprt := v.(Int)
				// skip failed reads
				if vprt.IsNil {
					continue
				}
				if failedOp, ok := failedMap[core.KVEntity{K: k, V: vprt}]; ok {
					anomalies = append(anomalies, G1Conflict{
						Op:      op,
						Mop:     mop,
						Writer:  failedOp,
						Element: k,
					})
				}
			}
		}
	}

	return anomalies
}

func g1bCases(history core.History) GCaseTp {
	interMap := make(map[core.KVEntity]core.Op)
	var anomalies []core.Anomaly

	for _, op := range core.FilterOkHistory(history) {
		valMap := make(map[string]Int)
		for _, mop := range *op.Value {
			if mop.IsWrite() {
				k, v := mop.GetKey(), mop.GetValue()
				vprt := v.(Int)
				if vprt.IsNil {
					panic("write value should not be nil")
				}
				if old, ok := valMap[k]; ok {
					interMap[core.KVEntity{K: k, V: old}] = op
				}
				valMap[k] = vprt
			}
		}
	}

	for _, op := range core.FilterOkHistory(history) {
		for _, mop := range *op.Value {
			if mop.IsRead() {
				k, v := mop.GetKey(), mop.GetValue()
				vprt := v.(Int)
				// skip failed reads
				if vprt.IsNil {
					continue
				}
				if interOp, ok := interMap[core.KVEntity{K: k, V: vprt}]; ok && op != interOp {
					anomalies = append(anomalies, G1Conflict{
						Op:      op,
						Mop:     mop,
						Writer:  interOp,
						Element: k,
					})
				}
			}
		}
	}

	return anomalies
}

type wrExplainResult struct {
	Typ   core.DependType
	Key   string
	Value core.MopValueType
}

// WRExplainResult creates a wrExplainResult
func WRExplainResult(key string, value core.MopValueType) wrExplainResult {
	return wrExplainResult{
		Typ:   core.WRDepend,
		Key:   key,
		Value: value,
	}
}

func (w wrExplainResult) Type() core.DependType {
	return core.WRDepend
}

// wwExplainer explains write-write dependencies
type wrExplainer struct{}

func (w *wrExplainer) ExplainPairData(a, b core.PathType) core.ExplainResult {
	var (
		writes = extWriteKeys(a)
		reads  = extReadKeys(b)
	)

	for k, wv := range writes {
		if rv, ok := reads[k]; ok && rv.Eq(wv) {
			return WRExplainResult(k, wv)
		}
	}

	return nil
}

func (w *wrExplainer) RenderExplanation(result core.ExplainResult, a, b string) string {
	if result.Type() != core.WRDepend {
		log.Fatalf("result type is not %s, type error", core.WWDepend)
	}
	er := result.(wrExplainResult)
	return fmt.Sprintf("%s read %v written by %s with value %s",
		b, er.Key, a, er.Value,
	)
}

// wwGraph analyzes write-write dependencies
func wrGraph(history core.History, _ ...interface{}) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	history = core.FilterOkHistory(history)
	var (
		extReadIndex  = extIndex(extReadKeys, history)
		extWriteIndex = extIndex(extWriteKeys, history)
		g             = core.NewDirectedGraph()
	)

	for k, rvals := range extReadIndex {
		for v, rops := range rvals {
			wvals, ok := extWriteIndex[k]
			if !ok {
				continue
			}
			wops, ok := wvals[v]
			if !ok {
				continue
			}
			if len(wops) == 0 {
				continue
			} else if len(wops) == 1 {
				var reads []core.Vertex
				for _, rop := range rops {
					reads = append(reads, core.Vertex{Value: rop})
				}
				g.LinkToAll(core.Vertex{Value: wops[0]}, reads, core.WR)
			} else {
				panic("multi ext write keys")
			}
		}
	}

	return nil, g, &wrExplainer{}
}

// getKeys get all keys from an op
func getKeys(op core.Op) []string {
	var (
		keys []string
		dup  = make(map[string]struct{})
	)
	for _, mop := range *op.Value {
		// don't infer crashed reads
		// but infer crashed writes
		if op.Type == core.OpTypeInfo && mop.IsRead() {
			continue
		}

		k := mop.GetKey()
		if _, ok := dup[k]; !ok {
			keys = append(keys, k)
			dup[k] = struct{}{}
		}
	}
	return keys
}

// getVersion starts by find first mop which read or write the given key
// read  => version is the read value
// write => go through the rest mops, version is the last value assigned to this key
func getVersion(k string, op core.Op) Int {
	r := NewNil()
	tp := core.MopTypeUnknown
	for _, mop := range *op.Value {
		if op.Type == core.OpTypeInfo && mop.IsRead() {
			continue
		}
		if k == mop.GetKey() {
			v := mop.GetValue()
			if tp == core.MopTypeUnknown {
				tp = mop.T
				r = v.(Int)
				if tp == core.MopTypeRead {
					return r
				}
			}
			if tp == core.MopTypeWrite && tp == mop.T {
				r = v.(Int)
			}
		}
	}
	return r
}

// transactionGraph2VersionGraphs runs based on realtime or process graph
// case1
// T1[wx1], T2[rx2wx3]
// T1[T2] => 1 => [2]
// case2
// T1[wx1rx2], T2[wx3wx4]
// T1[T2] => 2 => [4]
// case3
// T1[rx1], T2[wx2], T3[wx3], T4[rx4]
// T1[T2, T2], T2[T4], T3[T4] => 1 => [2, 3], 2 => [4], 3 => [4]
func transactionGraph2VersionGraphs(rel core.Rel, history core.History, graph *core.DirectedGraph) map[string]*core.DirectedGraph {
	gs := make(map[string]*core.DirectedGraph)
	cache := make(map[string]map[core.Op][]core.Op)
	var find func(key string, op core.Op) []core.Op
	find = func(key string, op core.Op) []core.Op {
		if _, ok := cache[key]; !ok {
			cache[key] = make(map[core.Op][]core.Op)
		}
		if ops, ok := cache[key][op]; ok {
			return ops
		}
		var ops []core.Op
		nexts, ok := graph.Outs[core.Vertex{Value: op}]
		if !ok {
			return ops
		}
		for vertex, rels := range nexts {
			hasRel := false
			for _, r := range rels {
				if r == rel {
					hasRel = true
				}
			}
			if !hasRel {
				continue
			}
			nextOp := vertex.Value.(core.Op)
			stopNext := false
		LOOP:
			for _, mop := range *nextOp.Value {
				if key == mop.GetKey() {
					if (mop.IsWrite() && (nextOp.Type == core.OpTypeOk || nextOp.Type == core.OpTypeInfo)) ||
						(mop.IsRead() && nextOp.Type == core.OpTypeOk) {
						stopNext = true
						break LOOP
					}
				}
			}

			if stopNext {
				cache[key][op] = []core.Op{nextOp}
				ops = append(ops, nextOp)
			} else {
				nextOps := find(key, nextOp)
				cache[key][op] = nextOps
				ops = append(ops, nextOps...)
			}
		}
		return ops
	}

	for _, op := range core.FilterOkOrInfoHistory(history) {
		keys := getKeys(op)

		for _, k := range keys {
			nexts := find(k, op)
			g, ok := gs[k]
			if !ok {
				g = core.NewDirectedGraph()
				gs[k] = g
			}
			selfV := getVersion(k, op)
			if selfV.IsNil {
				// should not be nill
				panic("self version should not be nil")
			}
			for _, next := range nexts {
				nextV := getVersion(k, next)
				if nextV.IsNil {
					continue
				}
				if selfV.Eq(nextV) {
					continue
				}
				g.Link(core.Vertex{Value: selfV}, core.Vertex{Value: nextV}, rel)
			}
		}
	}

	return gs
}

func initialStateVersionGraphs(history core.History) map[string]*core.DirectedGraph {
	vgs := make(map[string]*core.DirectedGraph)

	for _, op := range history {
		if op.Type == core.OpTypeFail || op.Type == core.OpTypeInvoke {
			continue
		}
		kvs := extWriteKeys(op)
		if op.Type == core.OpTypeOk {
			for k, v := range extReadKeys(op) {
				kvs[k] = v
			}
		}
		for k, v := range kvs {
			if _, ok := vgs[k]; !ok {
				vgs[k] = core.NewDirectedGraph()
			}
			init := core.Vertex{Value: NewNil()}
			if _, ok := vgs[k].Outs[init]; !ok {
				vgs[k].Link(init, core.Vertex{Value: v}, core.InitialState)
			}
		}
	}

	return vgs
}

func wfrVersionGraphs(history core.History) map[string]*core.DirectedGraph {
	vgs := make(map[string]*core.DirectedGraph)
	for _, op := range core.FilterOkHistory(history) {
		var (
			reads  = extReadKeys(op)
			writes = extWriteKeys(op)
		)
		for k, r := range reads {
			if r.IsNil {
				continue
			}
			w, ok := writes[k]
			if !ok {
				continue
			}
			if _, ok := vgs[k]; !ok {
				vgs[k] = core.NewDirectedGraph()
			}
			vgs[k].Link(core.Vertex{Value: r}, core.Vertex{Value: w}, core.WFR)
		}
	}
	return vgs
}

func sequentialKeysGraphs(history core.History) map[string]*core.DirectedGraph {
	_, graph, _ := core.ProcessGraph(history)
	return transactionGraph2VersionGraphs(core.Process, history, graph)
}

func linearizableKeysGraphs(history core.History) map[string]*core.DirectedGraph {
	_, graph, _ := core.RealtimeGraph(history)
	return transactionGraph2VersionGraphs(core.Realtime, history, graph)
}

func mergeGraphs(g1s, g2s map[string]*core.DirectedGraph, with func(...*core.DirectedGraph) *core.DirectedGraph) map[string]*core.DirectedGraph {
	gs := make(map[string]*core.DirectedGraph)
	for k, g1 := range g1s {
		if g2, ok := g2s[k]; ok {
			gs[k] = with(g1, g2)
		} else {
			gs[k] = g1
		}
	}
	for k, g2 := range g2s {
		if _, ok := gs[k]; !ok {
			gs[k] = g2
		}
	}
	return gs
}

type cyclicVersion struct {
	key     string
	scc     []Int
	sources []string
}

func (cyclicVersion) IAnomaly() {}

// String ...
func (c cyclicVersion) String() string {
	return fmt.Sprintf("(cyclicVersion) key: %s, scc: %v, sources: %v",
		c.key, c.scc, c.sources)
}

func cyclicVersionCases(versionGraphs map[string]*core.DirectedGraph) core.Anomalies {
	cases := make(core.Anomalies)

	for key, graph := range versionGraphs {
		var sources []string
		for _, outs := range graph.Outs {
			for _, rels := range outs {
				for _, rel := range rels {
					sources = append(sources, string(rel))
				}
			}
		}
		sources = core.Set(sources)
		sort.Strings(sources)
		for _, scc := range graph.StronglyConnectedComponents() {
			var iscc []Int
			for _, v := range scc.Vertices {
				iscc = append(iscc, v.Value.(Int))
			}
			cycleCase := cyclicVersion{
				key:     key,
				scc:     iscc,
				sources: sources,
			}
			cases.Merge(core.Anomalies{
				"cyclic-versions": []core.Anomaly{cycleCase},
			})
		}
	}
	return cases
}

func versionGraphs(history core.History, opts ...interface{}) (core.Anomalies, []string, map[string]*core.DirectedGraph) {
	opt := opts[0].(GraphOption)

	type analyzer struct {
		source   string
		analyzer func(history core.History) map[string]*core.DirectedGraph
	}

	analyzers := []analyzer{
		{
			source:   "initial-state-version-graphs",
			analyzer: initialStateVersionGraphs,
		},
	}

	if opt.LinearizableKeys {
		analyzers = append(analyzers, analyzer{
			source:   "linearizable-keys-graphs",
			analyzer: linearizableKeysGraphs,
		})
	}
	if opt.SequentialKeys {
		analyzers = append(analyzers, analyzer{
			source:   "sequential-keys-graphs",
			analyzer: sequentialKeysGraphs,
		})
	}
	if opt.WfrKeys {
		analyzers = append(analyzers, analyzer{
			source:   "wfr-version-graphs",
			analyzer: wfrVersionGraphs,
		})
	}

	var (
		anomalies = make(core.Anomalies)
		sources   []string
		gs        = make(map[string]*core.DirectedGraph)
	)
	for _, a := range analyzers {
		nextGs := mergeGraphs(gs, a.analyzer(history), core.DigraphUnion)

		cs := cyclicVersionCases(nextGs)
		if len(cs) != 0 {
			anomalies.Merge(cs)
		} else {
			sources = append(sources, a.source)
			gs = nextGs
		}
	}

	return anomalies, sources, gs
}

func extIndex(fn func(op core.Op) map[string]Int, history core.History) map[string]map[Int][]core.Op {
	res := make(map[string]map[Int][]core.Op)

	okHistory := core.FilterOkHistory(history)
	for _, op := range okHistory {
		extKV := fn(op)
		for k, v := range extKV {
			if _, ok := res[k]; !ok {
				res[k] = make(map[Int][]core.Op)
			}
			if _, ok := res[k][v]; !ok {
				res[k][v] = make([]core.Op, 0)
			}
			res[k][v] = append(res[k][v], op)
		}
	}

	return res
}

func versionGraph2TransactionGraph(key string, history core.History, versionGraph *core.DirectedGraph) *core.DirectedGraph {
	var (
		extReadIndex  = extIndex(extReadKeys, history)
		extWriteIndex = extIndex(extWriteKeys, history)
		g             = core.NewDirectedGraph()
	)

	for from, nexts := range versionGraph.Outs {
		v1 := from.Value.(Int)
		for next := range nexts {
			v2 := next.Value.(Int)
			keyReads, ok := extReadIndex[key]
			if !ok {
				keyReads = make(map[Int][]core.Op)
			}
			keyWrites, ok := extWriteIndex[key]
			if !ok {
				keyWrites = make(map[Int][]core.Op)
			}
			v1Reads, ok := keyReads[v1]
			if !ok {
				v1Reads = make([]core.Op, 0)
			}
			v1Writes, ok := keyWrites[v1]
			if !ok {
				v1Writes = make([]core.Op, 0)
			}
			v2Writes, ok := keyWrites[v2]
			if !ok {
				v2Writes = make([]core.Op, 0)
			}
			g.LinkAllToAll(core.NewVerticesFromOp(v1Writes), core.NewVerticesFromOp(v2Writes), core.WW)
			g.LinkAllToAll(core.NewVerticesFromOp(v1Reads), core.NewVerticesFromOp(v2Writes), core.RW)
			all := append(v1Reads, append(v1Writes, v2Writes...)...)
			g.UnLinkSelfEdges(core.NewVerticesFromOp(all))
		}
	}

	return g
}

func versionGraphs2TransactionGraph(history core.History, graphs map[string]*core.DirectedGraph) *core.DirectedGraph {
	var gs []*core.DirectedGraph
	for k, g := range graphs {
		gs = append(gs, versionGraph2TransactionGraph(k, history, g))
	}
	return core.DigraphUnion(gs...)
}

type wwExplainResult struct {
	Typ       core.DependType
	key       string
	prevValue interface{}
	value     interface{}
}

// WWExplainResult creates a wwExplainResult
func WWExplainResult(key string, prevValue, value core.MopValueType) wwExplainResult {
	return wwExplainResult{
		Typ:       core.WWDepend,
		key:       key,
		prevValue: prevValue,
		value:     value,
	}
}

func (wwExplainResult) Type() core.DependType {
	return core.WWDepend
}

// wwExplainer explains write-write dependencies
type wwExplainer struct {
	versionGraphs map[string]*core.DirectedGraph
}

func (w *wwExplainer) ExplainPairData(a, b core.PathType) core.ExplainResult {
	// if pair := explainPairData(a, b, core.MopTypeWrite, core.MopTypeWrite); pair != nil {
	// 	return WWExplainResult(pair.k, pair.prevValue, pair.value)
	// }
	k, prev, v := explainOpDeps(w.versionGraphs, extWriteKeys, a, extWriteKeys, b)
	if !prev.IsNil && !v.IsNil {
		return WWExplainResult(k, prev, v)
	}
	return nil
}

func (w *wwExplainer) RenderExplanation(result core.ExplainResult, a, b string) string {
	if result.Type() != core.WWDepend {
		log.Fatalf("result type is not %s, type error", core.WWDepend)
	}
	er := result.(wwExplainResult)
	return fmt.Sprintf("%s written %v with value %s written by %s with value %s",
		b, er.key, er.prevValue, a, er.value,
	)
}

type rwExplainResult struct {
	Typ       core.DependType
	key       string
	prevValue interface{}
	value     interface{}
}

// RWExplainResult creates a rwExplainResult
func RWExplainResult(key string, prevValue, value core.MopValueType) rwExplainResult {
	return rwExplainResult{
		Typ:       core.RWDepend,
		key:       key,
		prevValue: prevValue,
		value:     value,
	}
}

func (rwExplainResult) Type() core.DependType {
	return core.RWDepend
}

// wwExplainer explains write-write dependencies
type rwExplainer struct {
	versionGraphs map[string]*core.DirectedGraph
}

func (r *rwExplainer) ExplainPairData(a, b core.PathType) core.ExplainResult {
	k, prev, v := explainOpDeps(r.versionGraphs, extReadKeys, a, extWriteKeys, b)
	if !prev.IsNil || !v.IsNil {
		return RWExplainResult(k, prev, v)
	}
	return nil
}

func (*rwExplainer) RenderExplanation(result core.ExplainResult, a, b string) string {
	if result.Type() != core.RWDepend {
		log.Fatalf("result type is not %s, type error", core.RWDepend)
	}
	er := result.(rwExplainResult)
	return fmt.Sprintf("%s read %v with value %s written by %s with value %s",
		b, er.key, er.prevValue, a, er.value,
	)
}

// WWRWGraph ...
func WWRWGraph(history core.History, opts ...interface{}) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	anomalies, _, versionGraphs := versionGraphs(history, opts...)
	txnGraph := versionGraphs2TransactionGraph(history, versionGraphs)
	return anomalies, txnGraph, core.NewCombineExplainer([]core.DataExplainer{
		&wwExplainer{versionGraphs: versionGraphs},
		&rwExplainer{versionGraphs: versionGraphs},
	})
}

// graph combines wwGraph, wrGraph and rwGraph
func graph(history core.History, opts ...interface{}) (core.Anomalies, *core.DirectedGraph, core.DataExplainer) {
	return core.Combine(wrGraph, WWRWGraph)(history, opts...)
}

// Check checks append and read history for list_append
func Check(opts txn.Opts, history core.History, graphOpt GraphOption) txn.CheckResult {
	history = preProcessHistory(history)
	g1a := g1aCases(history)
	g1b := g1bCases(history)
	internal := internal(history)
	var analyzer core.Analyzer = graph
	additionalGraphs := txn.AdditionalGraphs(opts)
	if len(additionalGraphs) != 0 {
		analyzer = core.Combine(append([]core.Analyzer{analyzer}, additionalGraphs...)...)
	}

	checkResult := txn.Cycles(analyzer, history, graphOpt)
	anomalies := checkResult.Anomalies
	if len(g1a) != 0 {
		anomalies["G1a"] = g1a
	}
	if len(g1b) != 0 {
		anomalies["G1b"] = g1b
	}
	if len(internal) != 0 {
		anomalies["internal"] = internal
	}
	return txn.ResultMap(opts, anomalies)
}
