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

// Takes an options map, including a collection of expected consistency models
//  :consistency-models, a set of additional :anomalies, an analyzer function,
//  and a history. Analyzes the history and yields the analysis, plus an anomaly
//  map like {:G1c [...]}.
func cycles(analyzer core.Analyzer, history core.History) core.Anomalies {
	checkedResult := core.Check(analyzer, history)
	cases := CycleCases(checkedResult.Graph, checkedResult.Explainer, checkedResult.Sccs)
	for k, v := range cases {
		checkedResult.Anomalies[k] = v
	}
	return checkedResult.Anomalies
}

type CycleCase struct {
	Circle core.Circle
	Steps  []core.Step
	Type   string
}

func CycleCases(graph core.DirectedGraph, pairExplainer core.DataExplainer, sccs []core.SCC) map[string][]CycleCase {
	g := FilteredGraphs(graph)
	cases := map[string][]CycleCase{}
	for _, scc := range sccs {
		for _, v := range CycleCasesInScc(graph, g, pairExplainer, scc) {
			if _, e := cases[v.Type]; !e {
				cases[v.Type] = make([]CycleCase, 0)
			}
			cases[v.Type] = append(cases[v.Type], v)
		}
	}

	return cases
}

type FilterGraphFn = func(rels []core.Rel) *core.DirectedGraph

// FilteredGraphs receives a graph and a collection of relations, return a new Graph filtered to just those relationships
// Note: currently it use fork here, we can considering remove it.
func FilteredGraphs(graph core.DirectedGraph) FilterGraphFn {
	return func(rels []core.Rel) *core.DirectedGraph {
		return graph.Fork().FilterRelationships(rels)
	}
}

// CycleCasesInScc searches a single SCC for cycle anomalies
func CycleCasesInScc(graph core.DirectedGraph, filterGraph FilterGraphFn, explainer core.DataExplainer, scc core.SCC) []CycleCase {
	for _ := range CycleAnomalySpecs {
		panic("implement me")
	}
}

// CyclesWithDraw means "cycles!" in clojure.
func CyclesWithDraw() {
	panic("implement me")
}
