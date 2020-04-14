package txn

import (
	"github.com/pingcap/tipocket/pkg/elle/core"
)

type Opts struct {
}

type CheckResult struct {
	Valid            bool
	AnomalyTypes     []string
	Anomalies        core.Anomalies
	ImpossibleModels []interface{}
}

type CycleAnomalySpec struct {
	Rels      []core.Rel
	FirstRels []core.Rel
	RestRels  []core.Rel
	With      interface{} // seems only G-nonadjacent is used, it's a little complicated
	FilterEx  func(cycleCase CycleCase) bool
}

var CycleAnomalySpecs []CycleAnomalySpec

// Takes an options map, including a collection of expected consistency models
//  :consistency-models, a set of additional :anomalies, an analyzer function,
//  and a history. Analyzes the history and yields the analysis, plus an anomaly
//  map like {:G1c [...]}.
func Cycles(opts Opts, analyzer core.Analyzer, history core.History) core.Anomalies {
	panic("implement me")
}

type CycleExplainer struct{}

func (c *CycleExplainer) ExplainCycle(explainer core.DataExplainer, circle core.Circle) (core.Circle, []core.Step) {
	panic("impl me")
}

func (c *CycleExplainer) RenderCycleExplanation(explainer core.DataExplainer, circle core.Circle) string {
	panic("impl me")
}

type CycleCase struct {
	Circle core.Circle
	Steps  []core.Step
	Type   string
}

func CycleCases(graph core.DirectedGraph, pairExplainer core.DataExplainer, sccs []core.SCC) map[string]CycleCase {
	panic("impl me")
}

// FilteredGraphs receives a graph and a collection of relations, return a new Graph filtered to just those relationships
func FilteredGraphs(graph core.DirectedGraph, rels []interface{}) core.DirectedGraph {
	panic("impl me")
}

// CycleCasesInScc searches a single SCC for cycle anomalies
func CycleCasesInScc(graph core.DirectedGraph, explainer core.DataExplainer, scc core.SCC) []CycleCase {
	panic("impl me")
}

func init() {
	//CycleAnomalySpecs =
}
