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
func Cycles(analyzer core.Analyzer, history core.History) core.Anomalies {
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

// CycleCasesInScc searches a single SCC for cycle anomalies.
// TODO: add timeout logic.
func CycleCasesInScc(graph core.DirectedGraph, filterGraph FilterGraphFn, explainer core.DataExplainer, scc core.SCC) []CycleCase {
	var cases []CycleCase
	for _, v := range CycleAnomalySpecs {
		var runtimeGraph *core.DirectedGraph
		if v.Rels != nil {
			runtimeGraph = filterGraph(setKeys(v.Rels))
		} else {
			runtimeGraph = &graph
		}
		var cycle *core.Circle
		cycle = nil
		if v.With != nil {
			c := core.FindCycleWith(v.With, v.FilterPathState, *runtimeGraph, scc)
			cycle = &c
		} else if v.Rels != nil {
			c := core.FindCycle(*runtimeGraph, scc)
			cycle = &c
		} else {
			// Note: this requires find-cycle-starting-with
			s1 := filterGraph([]core.Rel{v.FirstRel})
			s2 := filterGraph(setKeys(v.RestRels))
			c := core.FindCycleStartingWith(*s1, *s2, scc)
			cycle = &c
		}

		if cycle != nil {
			explainerWrapper := CycleExplainerWrapper{}
			cycleCase := explainerWrapper.ExplainCycle(explainer, *cycle)
			if v.FilterEx != nil && !v.FilterEx(&cycleCase) {
				continue
			}
			cases = append(cases, cycleCase)
		}
	}
	return cases
}

// CyclesWithDraw means "cycles!" in clojure.
// TODO: This function contains some logic like draw, so I leave it unimplemented.
func CyclesWithDraw() {
	panic("implement me")
}

// Note: maybe cache this function is better?
func setKeys(m map[core.Rel]struct{}) []core.Rel {
	var rels []core.Rel
	for k, _ := range m {
		rels = append(rels, k)
	}
	return rels
}
