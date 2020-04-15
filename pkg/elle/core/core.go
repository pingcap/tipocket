package core

import "sort"

// Rel stands for relation in dependencies
type Rel string

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

// MergeAnomalies merges n Anomaly together.
func MergeAnomalies(anomalies ...Anomalies) Anomalies {
	panic("implement me")
}

// DataExplainer ...
type DataExplainer interface {
	// Given a pair of operations a and b, explains why b depends on a, in the
	//    form of a data structure. Returns `nil` if b does not depend on a.
	ExplainPairData() interface{}
	// Given a pair of operations, and short names for them, explain why b
	//  depends on a, as a string. `nil` indicates that b does not depend on a.
	RenderExplanation() string
}

// CombinedExplainer struct
type CombinedExplainer struct {
	Explainers []DataExplainer
}

// ExplainPairData find dependencies in a and b
func (c *CombinedExplainer) ExplainPairData() interface{} {
	return nil
}

// RenderExplanation render explanation result
func (c *CombinedExplainer) RenderExplanation() string {
	return ""
}

// CombineExplainer combines explainers into one
func CombineExplainer(explainers []DataExplainer) DataExplainer {
	return &CombinedExplainer{explainers}
}

// Combine composes multiple analyzers
func Combine(analyzers ...Analyzer) Analyzer {
	panic("implement me")
}

type Circle struct {
	// Eg. [2, 1, 2] means a circle: 2 -> 1 -> 2
	Path []interface{}
}

// TODO: refine me
type Step struct{}

// CycleExplainer provides the step-by-step explanation of the relationships between pairs of operations
type CycleExplainer struct {
}

func (c *CycleExplainer) explainCycle(explainer DataExplainer, circle Circle) (Circle, []Step) {
	panic("impl me")
}

func (c *CycleExplainer) renderCycleExplanation(explainer DataExplainer, circle Circle) string {
	panic("impl me")
}

// Analyzer is a function which takes a history and returns a {:graph, :explainer} map; e.g. realtime-graph.
type Analyzer func(history History) (DirectedGraph, DataExplainer)

// RealtimeGraph analyzes real-time
func RealtimeGraph(history History) (DirectedGraph, DataExplainer) {
	panic("implement me")
}

// ProcessExplainer ...
type ProcessExplainer struct{}

// ExplainPairData explain pair data
func (e ProcessExplainer) ExplainPairData() interface{} {
	panic("impl me")
}

// RenderExplanation render explanation
func (e ProcessExplainer) RenderExplanation() string {
	panic("impl me")
}

// ProcessOrder find dependencies of a process
func ProcessOrder(history History, process int) DirectedGraph {
	var (
		processHistory History
		graph          DirectedGraph
	)

	for _, op := range history {
		if op.Process == process {
			processHistory = append(processHistory, op)
		}
	}

	for i := 0; i < len(processHistory)-1; i++ {
		op1, op2 := processHistory[i], processHistory[i+1]
		graph.Link(Vertex{op1}, []Vertex{{op2}}, Process)
	}
	return *graph.Fork()
}

// ProcessGraph analyzes process
func ProcessGraph(history History) (DirectedGraph, DataExplainer) {
	var (
		okHistory = history.FilterType(OpTypeOk)
		processes map[int]struct{}
		graphs    []DirectedGraph
	)

	for _, op := range okHistory {
		if op.Process != 0 {
			if _, ok := processes[op.Process]; !ok {
				processes[op.Process] = struct{}{}
				graphs = append(graphs, ProcessOrder(okHistory, op.Process))
			}
		}
	}

	return *DigraphUion(graphs...), ProcessExplainer{}
}

// MonotonicKeyExplainer ...
type MonotonicKeyExplainer struct{}

// ExplainPairData explain pair data
func (e MonotonicKeyExplainer) ExplainPairData() interface{} {
	panic("impl me")
}

// RenderExplanation render explanation
func (e MonotonicKeyExplainer) RenderExplanation() string {
	panic("impl me")
}

// MonotonicKeyOrder find dependencies of a process
func MonotonicKeyOrder(history History, k string) DirectedGraph {
	var (
		val2ops map[int][]Op
		vals    []int
		graph   DirectedGraph
	)

	for _, op := range history {
		for _, mop := range op.Value {
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

	return graph
}

// MonotonicKeyGraph analyzes monotonic key
func MonotonicKeyGraph(history History) (DirectedGraph, DataExplainer) {
	var (
		okHistory = history.FilterType(OpTypeOk)
		keys      map[string]struct{}
		graphs    []DirectedGraph
	)

	// not sure if monotonic only works for read type mops
	for _, key := range okHistory.GetKeys(MopTypeRead) {
		if _, ok := keys[key]; !ok {
			keys[key] = struct{}{}
			graphs = append(graphs, MonotonicKeyOrder(okHistory, key))
		}
	}

	return *DigraphUion(graphs...), MonotonicKeyExplainer{}
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
	panic("implement me")
}
