package core

type Rel string

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

func CombineExplainer([]DataExplainer) DataExplainer {
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

//  A function which takes a history and returns a {:graph, :explainer, :anomalies} map; e.g. realtime-graph.
type Analyzer func(history History) (Anomalies, DirectedGraph, DataExplainer)

// RealtimeGraph analyzes real-time
func RealtimeGraph(history History) (Anomalies, DirectedGraph, DataExplainer) {
	panic("implement me")
}

// ProcessGraph analyzes process
func ProcessGraph(history History) (Anomalies, DirectedGraph, DataExplainer) {
	panic("implement me")
}

// Combine composes multiple analyzers
func Combine(analyzers ...Analyzer) Analyzer {
	panic("implement me")
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
