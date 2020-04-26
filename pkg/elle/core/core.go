package core

import (
	"sort"
)

// Rel stands for relation in dependencies
type Rel string

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

// Rel enums
const (
	Empty    Rel = ""
	WW       Rel = "ww"
	WR       Rel = "wr"
	RW       Rel = "rw"
	Process  Rel = "process"
	Realtime Rel = "realtime"
)

// Anomaly unifies all kinds of Anomalies, like G1a, G1b, dirty update etc.
type Anomaly interface{}
type Anomalies map[string]Anomaly

// Merge merges another anomalies
func (a Anomalies) Merge(another Anomalies) {
	for key, value := range another {
		a[key] = value
	}
}

// Analyzer is a function which takes a history and returns a {:graph, :explainer, :anomalies} map; e.g. realtime-graph.
type Analyzer func(history History) (Anomalies, *DirectedGraph, DataExplainer)

type PathType = Op

type Circle struct {
	// Eg. [2, 1, 2] means a circle: 2 -> 1 -> 2
	Path []PathType
}

// TODO: refine me
type Step struct {
	Result ExplainResult
}

// RealtimeGraph analyzes real-time.
func RealtimeGraph(history History) (Anomalies, *DirectedGraph, DataExplainer) {
	realtimeGraph := NewDirectedGraph()

	nextMap := make([]int, len(history), len(history))
	// build nextMap
	{
		processMap := map[int]int{}
		for i, v := range history {
			switch v.Type {
			case OpTypeNemesis, OpTypeFail, OpTypeInfo:
				nextMap[processMap[v.Process.MustGet()]] = i
				delete(processMap, v.Process.MustGet())
			case OpTypeInvoke:
				processMap[v.Process.MustGet()] = i
			case OpTypeOk:
				nextMap[processMap[v.Process.MustGet()]] = i
				nextMap[i] = processMap[v.Process.MustGet()]
				delete(processMap, v.Process.MustGet())
			}
		}
	}

	// build state machine
	var doneEvents = map[Op]struct{}{}
	for i := range history {
		v := &history[i]
		if !v.Process.Present() {
			continue
		}
		switch v.Type {
		case OpTypeNemesis, OpTypeFail, OpTypeInfo:
			continue
		case OpTypeInvoke:
			effectOp := history[nextMap[i]]
			for k := range doneEvents {
				realtimeGraph.Link(Vertex{Value: k}, Vertex{Value: effectOp}, Realtime)
			}
		case OpTypeOk:
			implied := opSet(realtimeGraph.In(Vertex{Value: history[i]}))
			doneEvents = setDel(doneEvents, implied)
			doneEvents[*v] = struct{}{}
		}
	}
	return nil, realtimeGraph, RealtimeExplainer{nextIndex: nextMap, historyReference: history}
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
	for k, _ := range delta {
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
func ProcessGraph(history History) (Anomalies, *DirectedGraph, DataExplainer) {
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
		graph.LinkAllToAll(xs, ys)
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

type CheckResult struct {
	Graph     DirectedGraph
	Explainer DataExplainer
	Cycles    []string // string explaining the cycle
	Sccs      []SCC
	Anomalies Anomalies
}

// Check receives analyzer and a history, returns a map of {graph, explainer, cycles, sccs, anomalies}
func Check(analyzer Analyzer, history History) CheckResult {
	g, explainer, circles, sccs, anomalies := checkHelper(analyzer, history)
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
// TODO: add the logic for anomalies.
func checkHelper(analyzer Analyzer, history History) (*DirectedGraph, DataExplainer, []string, []SCC, Anomalies) {
	// The sample program will first remove nemesis, but we will not leave nemesis here.
	anomalies, g, exp := analyzer(history)
	sccs := g.StronglyConnectedComponents()
	var cycles []string
	for _, scc := range sccs {
		cycles = append(cycles, explainSCC(g, CycleExplainer{}, exp, scc))
	}
	return g, exp, cycles, sccs, anomalies
}

// TODO: implement it.
func WriteCycles(cexp CycleExplainer, exp DataExplainer, dir, filename string, cycles []string) {
	return
}
