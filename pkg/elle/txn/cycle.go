package txn

import (
	"strings"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

type FilterExType = func(cycleCase *CycleCase) bool

type CycleAnomalySpecType struct {
	// A set of relationships which must intersect with every edge in the cycle.
	Rels map[core.Rel]struct{}

	// A set of relationships which must intersect with the first edge in the cycle.
	FirstRel core.Rel
	// A set of relationships which must intersect with remaining edges.
	RestRels map[core.Rel]struct{}

	// A predicate over a cycle explanation. We use this to restrict cycles to e.g. *just* G2 instead of G-single.
	FilterEx FilterExType

	With            func(n int, lastIsRw bool, rel core.Rel) (bool, int, bool) // seems only G-nonadjacent is used, it's a little complicated
	FilterPathState func(interface{}) bool
}

var CycleAnomalySpecs map[string]CycleAnomalySpecType
var CycleTypeNames map[string]struct{}
var UnknownAnomalyTypes map[string]struct{}

// Anomaly types involving realtime edges.
var RealtimeAnalysisTypes map[string]struct{}

var ProcessAnalysisTypes map[string]struct{}

func fromRels(rels ...core.Rel) CycleAnomalySpecType {
	return fromRelsWithFilter(nil, rels...)
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

func filterClosure(required string) FilterExType {
	return func(cycleCase *CycleCase) bool {
		return cycleCase.Type == required
	}
}

func init() {
	CycleAnomalySpecs = map[string]CycleAnomalySpecType{
		"G0":            fromRels(core.WW),
		"G1c":           fromFirstRelAndRest(core.WR, core.WW, core.WR),
		"G-single":      fromFirstRelAndRest(core.RW, core.WW, core.WR),
		"G-nonadjacent": fromRels(core.WW, core.WR, core.RW),

		"G2-item":          fromFirstRelAndRestWithFilter(filterClosure("G2-item"), core.RW, core.WR, core.RW, core.WW),
		"G0-process":       fromRelsWithFilter(filterClosure("G0-process"), core.WW, core.Process),
		"G1c-process":      fromFirstRelAndRestWithFilter(filterClosure("G1c-process"), core.RW, core.WW, core.WR, core.Process),
		"G-single-process": fromFirstRelAndRestWithFilter(filterClosure("G-single-process"), core.RW, core.WW, core.WR, core.Process),
		"G2-item-process":  fromFirstRelAndRestWithFilter(filterClosure("G2-item-process"), core.RW, core.WW, core.WR, core.RW, core.Process),

		// realtime

		"G0-realtime":       fromRelsWithFilter(filterClosure("G0-realtime"), core.WW, core.Realtime),
		"G1c-realtime":      fromFirstRelAndRestWithFilter(filterClosure("G1c-realtime"), core.WR, core.WW, core.WR, core.Realtime),
		"G-single-realtime": fromFirstRelAndRestWithFilter(filterClosure("G-single-realtime"), core.RW, core.WW, core.WR, core.Realtime),
		"G2-item-realtime":  fromFirstRelAndRestWithFilter(filterClosure("G2-item-realtime"), core.RW, core.WW, core.WR, core.Realtime, core.RW),
	}

	v := CycleAnomalySpecs["G-nonadjacent"]
	v.With = NonadjacentRW
	v.FilterPathState = func(p interface{}) bool {
		// TODO: we need more than one rw edge for this to count; otherwise it's G-single.
		return true
	}

	CycleTypeNames = map[string]struct{}{
		"G-nonadjacent-process":  {},
		"G-nonadjacent-realtime": {},
	}

	for k, _ := range CycleAnomalySpecs {
		CycleTypeNames[k] = struct{}{}
	}

	UnknownAnomalyTypes = map[string]struct{}{
		"empty-transaction-graph": {},
		"cycle-search-timeout":    {},
	}

	ProcessAnalysisTypes = map[string]struct{}{}
	RealtimeAnalysisTypes = map[string]struct{}{}
	for k, _ := range CycleAnomalySpecs {
		if strings.Contains(k, "process") {
			ProcessAnalysisTypes[k] = struct{}{}
		}
		if strings.Contains(k, "realtime") {
			RealtimeAnalysisTypes[k] = struct{}{}
		}
	}
}

// CycleExplainWrapper is a ICycleExplainer, it's also a wrapper for core.
type CycleExplainerWrapper struct{}

func (c CycleExplainerWrapper) ExplainCycle(pairExplainer core.DataExplainer, circle core.Circle) CycleCase {
	panic("implement me")
}

func (c CycleExplainerWrapper) RenderCycleExplanation(explainer core.DataExplainer, circle core.Circle) string {
	exp := core.CycleExplainer{}
	return exp.RenderCycleExplanation(explainer, circle)
}

// NonadjacentRW is an strange helper function. It returns (valid, rw-count, current-is-rw).
// when initialize, please provide [0, false].
// This fn ensures that no :rw is next to another by testing successive edge
//  types. In addition, we ensure that the first edge in the cycle is not an rw.
//  Cycles must have at least two edges, and in order for no two rw edges to be
//  adjacent, there must be at least one non-rw edge among them. This constraint
//  ensures a sort of boundary condition for the first and last nodes--even if
//  the last edge is rw, we don't have to worry about violating the nonadjacency
//  property when we jump to the first.
func NonadjacentRW(n int, lastIsRw bool, rel core.Rel) (bool, int, bool) {
	if rel != core.RW {
		return true, n, lastIsRw
	} else {
		return lastIsRw, n + 1, true
	}
}

type AdditionalGraphOption struct {
	ConsistencyModels []core.ConsistencyModelName
	Anomalies         []string
}

type GraphFunc = func(core.History) (core.Anomalies, core.DirectedGraph, core.DataExplainer)

// AdditionalGraphs determines what additional graphs we'll need to consider for this analysis.
func AdditionalGraphs(opts AdditionalGraphOption, additionalGraphs []GraphFunc) []GraphFunc {
	ats := reportableAnomalyTypes(opts.ConsistencyModels, opts.Anomalies)
	var graphFn GraphFunc
	if hasIntersection(ats, RealtimeAnalysisTypes) {
		graphFn = core.RealtimeGraph
	} else if hasIntersection(ats, ProcessAnalysisTypes) {
		graphFn = core.ProcessGraph
	} else {
		return additionalGraphs
	}
	return append(additionalGraphs, graphFn)
}

// Anomalies worth reporting on, even if they don't cause the test to fail.
func reportableAnomalyTypes(cm []core.ConsistencyModelName, anomalies []string) map[string]struct{} {
	dest := prohibitAnomalyTypes(cm, anomalies)
	union(dest, UnknownAnomalyTypes)
	return dest
}

func prohibitAnomalyTypes(cm []core.ConsistencyModelName, anomalies []string) map[string]struct{} {
	a1 := core.AllAnomaliesImplying(cm)
	a2 := core.AnomaliesProhibitedBy(anomalies)
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
	for k, _ := range unionSrc {
		dest[k] = struct{}{}
	}
}

func hasIntersection(m1, m2 map[string]struct{}) bool {
	for k, _ := range m2 {
		_, e := m1[k]
		if e {
			return true
		}
	}
	return false
}
