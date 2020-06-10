package txn

import (
	"fmt"
	"strings"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

// FilterExType is a predication on a cycle case
type FilterExType = func(cycleCase *core.CycleExplainerResult) bool

// CycleAnomalySpecType specifies different anomalies
type CycleAnomalySpecType struct {
	// A set of relationships which must intersect with every edge in the cycle.
	Rels map[core.Rel]struct{}

	// A set of relationships which must intersect with the first edge in the cycle.
	FirstRel core.Rel
	// A set of relationships which must intersect with remaining edges.
	RestRels map[core.Rel]struct{}

	// A predicate over a cycle explanation. We use this to restrict cycles to e.g. *just* G2 instead of G-single.
	FilterEx FilterExType

	// A predicate over a cycle
	With core.CyclePredicate
}

// CycleAnomalySpecs defines anomaly specs
var CycleAnomalySpecs map[string]CycleAnomalySpecType

// CycleTypeNames ...
var CycleTypeNames map[string]struct{}

// UnknownAnomalyTypes ...
var UnknownAnomalyTypes map[string]struct{}

// RealtimeAnalysisTypes saves types involving realtime edges.
var RealtimeAnalysisTypes map[string]struct{}

// ProcessAnalysisTypes saves types involving process edges
var ProcessAnalysisTypes map[string]struct{}

func fromRels(rels ...core.Rel) CycleAnomalySpecType {
	return fromRelsWithFilter(nil, rels...)
}

func fromRelsAndWith(with core.CyclePredicate, rels ...core.Rel) CycleAnomalySpecType {
	r := fromRels(rels...)
	r.With = with
	return r
}

func fromRelsWithFilter(filter FilterExType, rels ...core.Rel) CycleAnomalySpecType {
	relsSet := map[core.Rel]struct{}{}
	for _, v := range rels {
		relsSet[v] = struct{}{}
	}
	return CycleAnomalySpecType{
		Rels:     relsSet,
		FilterEx: filter,
	}
}

func fromFirstRelAndRest(first core.Rel, rests ...core.Rel) CycleAnomalySpecType {
	return fromFirstRelAndRestWithFilter(nil, first, rests...)
}

func fromFirstRelAndRestWithFilter(filter FilterExType, first core.Rel, rests ...core.Rel) CycleAnomalySpecType {
	restSet := map[core.Rel]struct{}{}
	for _, v := range rests {
		restSet[v] = struct{}{}
	}
	return CycleAnomalySpecType{
		FirstRel: first,
		RestRels: restSet,
		FilterEx: filter,
	}
}

func buildFilterExByType(required string) FilterExType {
	return func(cr *core.CycleExplainerResult) bool {
		return string(cr.Typ) == required
	}
}

// nonadjacentRW ensures that no :rw is next to another by testing successive edge types.
// In addition, we ensure that the first edge in the cycle is not an rw.
// And we need more than one rw edge for this to count, otherwise it's G-single
func nonadjacentRW(trace []core.CycleTrace) bool {
	if len(trace) < 2 {
		return false
	}
	// ensure that the first edge in the cycle is not an rw
	lastIsRw := true
	rwCount := 0
	for _, path := range trace {
		rw := len(path.Rels) == 1 && path.Rels[0] == core.RW
		if lastIsRw && rw {
			return false
		}
		if rw {
			rwCount++
		}
		lastIsRw = rw
	}
	return rwCount > 1
}

func init() {
	CycleAnomalySpecs = map[string]CycleAnomalySpecType{
		"G0":               fromRels(core.WW),
		"G1c":              fromFirstRelAndRest(core.WR, core.WW, core.WR),
		"G-single":         fromFirstRelAndRest(core.RW, core.WW, core.WR),
		"G-nonadjacent":    fromRelsAndWith(nonadjacentRW, core.WW, core.WR, core.RW),
		"G2-item":          fromFirstRelAndRestWithFilter(buildFilterExByType("G2-item"), core.RW, core.WR, core.RW, core.WW),
		"G0-process":       fromRelsWithFilter(buildFilterExByType("G0-process"), core.WW, core.Process),
		"G1c-process":      fromFirstRelAndRestWithFilter(buildFilterExByType("G1c-process"), core.WR, core.WW, core.WR, core.Process),
		"G-single-process": fromFirstRelAndRestWithFilter(buildFilterExByType("G-single-process"), core.RW, core.WW, core.WR, core.Process),
		"G2-item-process":  fromFirstRelAndRestWithFilter(buildFilterExByType("G2-item-process"), core.RW, core.WW, core.WR, core.RW, core.Process),
		// realtime
		"G0-realtime":       fromRelsWithFilter(buildFilterExByType("G0-realtime"), core.WW, core.Realtime),
		"G1c-realtime":      fromFirstRelAndRestWithFilter(buildFilterExByType("G1c-realtime"), core.WR, core.WW, core.WR, core.Realtime),
		"G-single-realtime": fromFirstRelAndRestWithFilter(buildFilterExByType("G-single-realtime"), core.RW, core.WW, core.WR, core.Realtime),
		"G2-item-realtime":  fromFirstRelAndRestWithFilter(buildFilterExByType("G2-item-realtime"), core.RW, core.WW, core.WR, core.Realtime, core.RW),
	}

	CycleTypeNames = map[string]struct{}{
		"G-nonadjacent-process":  {},
		"G-nonadjacent-realtime": {},
	}

	for k := range CycleAnomalySpecs {
		CycleTypeNames[k] = struct{}{}
	}

	UnknownAnomalyTypes = map[string]struct{}{
		"empty-transaction-graph": {},
		"cycle-search-timeout":    {},
	}

	ProcessAnalysisTypes = map[string]struct{}{}
	RealtimeAnalysisTypes = map[string]struct{}{}
	for k := range CycleTypeNames {
		if strings.Contains(k, "process") {
			ProcessAnalysisTypes[k] = struct{}{}
		}
		if strings.Contains(k, "realtime") {
			RealtimeAnalysisTypes[k] = struct{}{}
		}
	}
}

// CycleExplainerWrapper is a ICycleExplainer, it's also a wrapper for core.
type CycleExplainerWrapper struct{}

// ExplainCycle ...
func (c CycleExplainerWrapper) ExplainCycle(pairExplainer core.DataExplainer, circle core.Circle) core.CycleExplainerResult {
	ce := core.CycleExplainer{}
	ex := ce.ExplainCycle(pairExplainer, circle)
	steps := ex.Steps
	typeFrequencies := make(map[core.DependType]int)
	for _, step := range ex.Steps {
		t := step.Result.Type()
		if _, ok := typeFrequencies[t]; !ok {
			typeFrequencies[t] = 0
		}
		typeFrequencies[t]++
	}
	realtime := typeFrequencies[core.RealtimeDepend]
	process := typeFrequencies[core.ProcessDepend]
	ww := typeFrequencies[core.WWDepend]
	wr := typeFrequencies[core.WRDepend]
	rw := typeFrequencies[core.RWDepend]
	var rwAdj bool
	var lastType = steps[len(steps)-1].Result.Type()
	for _, step := range steps {
		if lastType == core.RWDepend && step.Result.Type() == core.RWDepend {
			rwAdj = true
			break
		}
		lastType = step.Result.Type()
	}
	var dataDepType string
	if rw == 1 {
		dataDepType = "G-single"
	} else if 1 < rw {
		if rwAdj {
			dataDepType = "G2-item"
		} else {
			dataDepType = "G-nonadjacent"
		}
	} else if 0 < wr {
		dataDepType = "G1c"
	} else if 0 < ww {
		dataDepType = "G0"
	} else {
		panic(fmt.Sprintf("Don't know how to classify: %+v", ex))
	}
	var subtype string
	if 0 < realtime {
		subtype = "-realtime"
	} else if 0 < process {
		subtype = "-process"
	}
	return core.CycleExplainerResult{
		Circle: ex.Circle,
		Steps:  ex.Steps,
		Typ:    fmt.Sprintf("%s%s", dataDepType, subtype),
	}
}

// RenderCycleExplanation ...
func (c CycleExplainerWrapper) RenderCycleExplanation(explainer core.DataExplainer, cr core.CycleExplainerResult) string {
	exp := core.CycleExplainer{}
	return exp.RenderCycleExplanation(explainer, cr)
}

// AdditionalGraphs determines what additional graphs we'll need to consider for this analysis.
func AdditionalGraphs(opts Opts) []core.Analyzer {
	ats := reportableAnomalyTypes(opts.ConsistencyModels, opts.Anomalies)
	var graphFn core.Analyzer
	if hasIntersection(ats, RealtimeAnalysisTypes) {
		graphFn = core.RealtimeGraph
	} else if hasIntersection(ats, ProcessAnalysisTypes) {
		graphFn = core.ProcessGraph
	} else {
		return opts.AdditionalGraphs
	}
	return append(opts.AdditionalGraphs, graphFn)
}

// Anomalies worth reporting on, even if they don't cause the test to fail.
func reportableAnomalyTypes(cm []core.ConsistencyModelName, anomalies []string) map[string]struct{} {
	dest := prohibitedAnomalyTypes(cm, anomalies)
	union(dest, UnknownAnomalyTypes)
	return dest
}

func prohibitedAnomalyTypes(cm []core.ConsistencyModelName, anomalies []string) map[string]struct{} {
	if cm == nil {
		cm = append(cm, "strict-serializable")
	}
	a1 := core.AllAnomaliesImplying(anomalies)
	a2 := core.AnomaliesProhibitedBy(cm)
	dest := compactAnomalies(a1...)
	union(dest, compactAnomalies(a2...))
	return dest
}

func compactAnomalies(anomalies ...string) map[string]struct{} {
	ret := map[string]struct{}{}
	for _, v := range anomalies {
		ret[v] = struct{}{}
	}
	return ret
}

func union(dest map[string]struct{}, unionSrc map[string]struct{}) {
	for k := range unionSrc {
		dest[k] = struct{}{}
	}
}

func hasIntersection(m1, m2 map[string]struct{}) bool {
	for k := range m2 {
		_, e := m1[k]
		if e {
			return true
		}
	}
	return false
}
