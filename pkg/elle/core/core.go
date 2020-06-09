package core

import (
	"log"
	"sort"
)

// Rel stands for relation in dependencies
type Rel string

// RelSet type aliases []Rel
type RelSet []Rel

func (r RelSet) Len() int {
	return len(r)
}

func (r RelSet) Less(i, j int) bool {
	return r[i] < r[j]
}

func (r RelSet) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

// Append returns a new RelSet contains the old and appended relations
func (r RelSet) Append(rels map[Rel]struct{}) (rs RelSet) {
	set := make(map[Rel]struct{})
	for _, rel := range r {
		set[rel] = struct{}{}
	}
	for rel := range rels {
		set[rel] = struct{}{}
	}
	for rel := range set {
		rs = append(rs, rel)
	}
	return
}

// Rel enums
const (
	Empty        Rel = ""
	WW           Rel = "ww"
	WR           Rel = "wr"
	RW           Rel = "rw"
	Process      Rel = "process"
	Realtime     Rel = "realtime"
	ExtKey       Rel = "ext-key"
	Version      Rel = "version"
	InitialState Rel = "initial-state"
	WFR          Rel = "write-follow-read"
	// Note: currently we don't support MonotonicKey
	MonotonicKey Rel = "monotonic-key"
)

// Anomaly unifies all kinds of Anomalies, like G1a, G1b, dirty update etc.
type Anomaly interface {
	IAnomaly()
}

// Anomalies groups anomalies by there names
type Anomalies map[string][]Anomaly

// Merge merges another anomalies
func (a Anomalies) Merge(another Anomalies) {
	for key, value := range another {
		if _, ok := a[key]; !ok {
			a[key] = value
		} else {
			a[key] = append(a[key], value...)
		}
	}
}

// SelectKeys selects specified keys and return a new Anomalies
func (a Anomalies) SelectKeys(anomalyNames map[string]struct{}) Anomalies {
	anomalies := make(Anomalies)
	for name := range anomalyNames {
		if value, ok := a[name]; ok {
			anomalies[name] = value
		}
	}
	return anomalies
}

// Keys returns all keys of Anomalies
func (a Anomalies) Keys() (keys []string) {
	for key := range a {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return
}

// Analyzer is a function which takes a history and returns a {:graph, :explainer, :anomalies} map; e.g. realtime-graph.
type Analyzer func(history History, opts ...interface{}) (Anomalies, *DirectedGraph, DataExplainer)

// PathType type aliases Op
type PathType = Op

// IndexOfMop returns the index of mop in op
func (op PathType) IndexOfMop(mop Mop) int {
	for idx, m := range *op.Value {
		if m.IsEqual(mop) {
			return idx
		}
	}
	return -1
}

// Circle ...
type Circle struct {
	// Eg. [2, 1, 2] means a circle: 2 -> 1 -> 2
	Path []PathType
}

// NewCircle returns a Circle from a Vertex slice
func NewCircle(vertices []Vertex) *Circle {
	if len(vertices) == 0 {
		return nil
	}
	if len(vertices) < 3 || vertices[0] != vertices[len(vertices)-1] {
		panic("there isn't a cycle, the code may has bug")
	}
	c := &Circle{Path: make([]PathType, 0)}
	for _, vertex := range vertices {
		c.Path = append(c.Path, vertex.Value.(PathType))
	}
	return c
}

// Step saves a explain result of one step of a cycle
type Step struct {
	Result ExplainResult
}

// RealtimeGraph analyzes real-time.
func RealtimeGraph(history History, _ ...interface{}) (Anomalies, *DirectedGraph, DataExplainer) {
	realtimeGraph := NewDirectedGraph()

	pair := make(map[Op]Op)
	{
		invocations := map[int]Op{}
		for _, v := range history {
			process := v.Process.GetOr(AnonymousMagicNumber)

			switch v.Type {
			case OpTypeInvoke:
				invocations[process] = v
			case OpTypeInfo:
				invocation, e := invocations[process]
				if e {
					pair[invocation] = v
					pair[v] = invocation
					delete(invocations, process)
				} else {
					invocations[process] = v
				}
			case OpTypeOk, OpTypeFail:
				invocation, e := invocations[process]
				if !e {
					log.Fatalf("cannot find the invocation of %s, the code may has bug", v)
				}
				pair[invocation] = v
				pair[v] = invocation
				delete(invocations, process)
			}
		}
	}

	// build state machine
	var doneEvents = map[Op]struct{}{}
	for i := range history {
		op := history[i]
		switch op.Type {
		case OpTypeInvoke:
			pairOp := pair[op]
			for k := range doneEvents {
				realtimeGraph.Link(Vertex{Value: k}, Vertex{Value: pairOp}, Realtime)
			}
		case OpTypeOk:
			implied := opSet(realtimeGraph.In(Vertex{Value: op}))
			doneEvents = setDel(doneEvents, implied)
			doneEvents[op] = struct{}{}
		case OpTypeFail, OpTypeInfo:
			continue
		}
	}
	return nil, realtimeGraph, RealtimeExplainer{pair: pair}
}

func opSet(vertex []Vertex) map[Op]struct{} {
	dataMap := map[Op]struct{}{}
	for _, v := range vertex {
		op, e := v.Value.(Op)
		if !e {
			continue
		}
		dataMap[op] = struct{}{}
	}
	return dataMap
}

func setDel(origin, delta map[Op]struct{}) map[Op]struct{} {
	for k := range delta {
		delete(origin, k)
	}
	return origin
}

// ProcessOrder find dependencies of a process
func ProcessOrder(history History, process int) *DirectedGraph {
	var (
		processHistory History
		graph          *DirectedGraph = NewDirectedGraph()
	)

	for _, op := range history {
		if op.Process.Present() && op.Process.MustGet() == process {
			processHistory = append(processHistory, op)
		}
	}

	for i := 0; i < len(processHistory)-1; i++ {
		op1, op2 := processHistory[i], processHistory[i+1]
		graph.Link(Vertex{op1}, Vertex{op2}, Process)
	}
	// TODO: make clear why this failed
	//return *graph.Fork()
	return graph
}

// ProcessGraph analyzes process
func ProcessGraph(history History, _ ...interface{}) (Anomalies, *DirectedGraph, DataExplainer) {
	var (
		okHistory = history.FilterType(OpTypeOk)
		processes = map[int]struct{}{}
		graphs    []*DirectedGraph
	)

	for _, op := range okHistory {
		if op.Process.Present() {
			if _, ok := processes[op.Process.MustGet()]; !ok {
				processes[op.Process.MustGet()] = struct{}{}
				graphs = append(graphs, ProcessOrder(okHistory, op.Process.MustGet()))
			}
		}
	}

	return nil, DigraphUnion(graphs...), ProcessExplainer{}
}

// MonotonicKeyOrder find dependencies of a process
func MonotonicKeyOrder(history History, k string) *DirectedGraph {
	var (
		val2ops = map[int][]Op{}
		vals    []int
		graph   DirectedGraph
	)

	for _, op := range history {
		for _, mop := range *op.Value {
			if mop.GetKey() != k {
				continue
			}
			// ignore list val type
			if mopVal, ok := mop.GetValue().(int); ok {
				if ops, ok := val2ops[mopVal]; ok {
					val2ops[mopVal] = append(ops, op)
				} else {
					val2ops[mopVal] = []Op{op}
					vals = append(vals, mopVal)
				}
			}
			// an operation needs to be record once only
			break
		}
	}

	sort.Ints(vals)
	for i := 0; i < len(vals)-1; i++ {
		var (
			xs []Vertex
			ys []Vertex
		)
		for _, x := range val2ops[vals[i]] {
			xs = append(xs, Vertex{x})
		}
		for _, y := range val2ops[vals[i]] {
			ys = append(ys, Vertex{y})
		}
		graph.LinkAllToAll(xs, ys, MonotonicKey)
	}

	return &graph
}

// MonotonicKeyGraph analyzes monotonic key
func MonotonicKeyGraph(history History) (Anomalies, *DirectedGraph, DataExplainer) {
	var (
		okHistory = history.FilterType(OpTypeOk)
		keys      = map[string]struct{}{}
		graphs    []*DirectedGraph
	)

	// not sure if monotonic only works for read type mops
	for _, key := range okHistory.GetKeys(MopTypeRead) {
		if _, ok := keys[key]; !ok {
			keys[key] = struct{}{}
			graphs = append(graphs, MonotonicKeyOrder(okHistory, key))
		}
	}

	return nil, DigraphUnion(graphs...), MonotonicKeyExplainer{}
}

// CheckResult records the check result
type CheckResult struct {
	Graph     DirectedGraph
	Explainer DataExplainer
	Cycles    []string // string explaining the cycle
	Sccs      []SCC
	Anomalies Anomalies
}

// Check receives analyzer and a history, returns a map of {graph, explainer, cycles, sccs, anomalies}
func Check(analyzer Analyzer, history History, opts ...interface{}) CheckResult {
	g, explainer, circles, sccs, anomalies := checkHelper(analyzer, history, opts...)
	WriteCycles(CycleExplainer{}, explainer, "", "", circles)
	return CheckResult{
		Graph:     *g,
		Explainer: explainer,
		Cycles:    circles,
		Sccs:      sccs,
		Anomalies: anomalies,
	}
}

// checkHelper is `check-` in original code.
func checkHelper(analyzer Analyzer, history History, opts ...interface{}) (*DirectedGraph, DataExplainer, []string, []SCC, Anomalies) {
	// The sample program will first remove nemesis, but we will not leave nemesis here.
	anomalies, g, exp := analyzer(history, opts...)
	sccs := g.StronglyConnectedComponents()
	var cycles []string
	for _, scc := range sccs {
		cycles = append(cycles, explainSCC(g, CycleExplainer{}, exp, scc))
	}
	if g.IsEmpty() {
		anomalies["empty-transaction-graph"] = []Anomaly{}
	}

	return g, exp, cycles, sccs, anomalies
}

// WriteCycles ...
// TODO: implement it.
func WriteCycles(cexp CycleExplainer, exp DataExplainer, dir, filename string, cycles []string) {
	return
}
