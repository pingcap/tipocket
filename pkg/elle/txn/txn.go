package txn

import (
	"encoding/json"
	"log"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

// Opts contains consistencyModels, Anomalies and AdditionalGraphs etc
type Opts struct {
	// define what specific anomalies and consistency models to look for
	ConsistencyModels []core.ConsistencyModelName
	Anomalies         []string
	AdditionalGraphs  []core.Analyzer
}

// CheckResult records the check result
type CheckResult struct {
	// valid? true | :unknown | false
	IsUnknown        bool           `json:"is_unknown"`
	Valid            bool           `json:"valid"`
	AnomalyTypes     []string       `json:"anomaly_types"`
	Anomalies        core.Anomalies `json:"anomalies"`
	ImpossibleModels []interface{}  `json:"impossible_models"`
	Not              []string       `json:"not"`
	AlsoNot          []string       `json:"also_not"`
}

func (c CheckResult) Error() string {
	data, err := json.Marshal(c)
	if err != nil {
		log.Fatalf("failed to marshal: %v", err)
	}
	return string(data)
}

// Cycles takes an options map, including a collection of expected consistency models
//  :consistency-models, a set of additional :anomalies, an analyzer function,
//  and a history. Analyzes the history and yields the analysis, plus an anomaly
//  map like {:G1c [...]}.
func Cycles(analyzer core.Analyzer, history core.History) core.CheckResult {
	checkedResult := core.Check(analyzer, history)
	cases := CycleCases(checkedResult.Graph, checkedResult.Explainer, checkedResult.Sccs)
	for k, v := range cases {
		checkedResult.Anomalies[k] = v
	}
	return checkedResult
}

// CycleCases finds anomaly cases and group them by there name
func CycleCases(graph core.DirectedGraph, pairExplainer core.DataExplainer, sccs []core.SCC) map[string][]core.Anomaly {
	g := FilteredGraphs(graph)
	cases := map[string][]core.Anomaly{}
	for _, scc := range sccs {
		for _, v := range CycleCasesInScc(graph, g, pairExplainer, scc) {
			if _, e := cases[string(v.Typ)]; !e {
				cases[string(v.Typ)] = make([]core.Anomaly, 0)
			}
			cases[string(v.Typ)] = append(cases[string(v.Typ)], v)
		}
	}
	return cases
}

// FilterGraphFn ...
type FilterGraphFn = func(rels []core.Rel) *core.DirectedGraph

// CycleCasesInScc searches a single SCC for cycle anomalies.
// TODO: add timeout logic.
func CycleCasesInScc(graph core.DirectedGraph, filterGraph FilterGraphFn, explainer core.DataExplainer, scc core.SCC) []core.CycleExplainerResult {
	var cases []core.CycleExplainerResult
	for cn, v := range CycleAnomalySpecs {
		_ = cn
		var runtimeGraph *core.DirectedGraph
		if v.Rels != nil {
			runtimeGraph = filterGraph(setKeys(v.Rels))
		} else {
			runtimeGraph = &graph
		}
		var cycle *core.Circle
		cycle = nil
		if v.With != nil {
			c := core.FindCycleWith(runtimeGraph, scc, v.With)
			cycle = core.NewCircle(c)
		} else if v.Rels != nil {
			c := core.FindCycle(runtimeGraph, scc)
			cycle = core.NewCircle(c)
		} else {
			// TODO(mahjonp): need review
			// Note: this requires find-cycle-starting-with
			//s1 := filterGraph([]core.Rel{v.FirstRel})
			//s2 := filterGraph(setKeys(v.RestRels))
			filteredGraph := filterGraph(core.RelSet([]core.Rel{v.FirstRel}).Append(v.RestRels))
			c := core.FindCycleStartingWith(filteredGraph, scc, v.FirstRel, core.RelSet{}.Append(v.RestRels))
			cycle = core.NewCircle(c)
		}

		if cycle != nil {
			explainerWrapper := CycleExplainerWrapper{}
			ex := explainerWrapper.ExplainCycle(explainer, *cycle)
			if v.FilterEx != nil && !v.FilterEx(&ex) {
				continue
			}
			cases = append(cases, ex)
		}
	}
	return cases
}

// CyclesWithDraw means "cycles!" in clojure.
// TODO: This function contains some logic like draw, so I leave it unimplemented.
func CyclesWithDraw() {
	panic("implement me")
}
