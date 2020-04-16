package txn

import "github.com/pingcap/tipocket/pkg/elle/core"

type FilterExType = func(cycleCase CycleCase) bool

type CycleAnomalySpecType struct {
	// A set of relationships which must intersect with every edge in the cycle.
	Rels map[core.Rel]struct{}

	// A set of relationships which must intersect with the first edge in the cycle.
	FirstRel core.Rel
	// A set of relationships which must intersect with remaining edges.
	RestRels map[core.Rel]struct{}

	// A predicate over a cycle explanation. We use this to restrict cycles to e.g. *just* G2 instead of G-single.
	FilterEx FilterExType

	With      interface{} // seems only G-nonadjacent is used, it's a little complicated
}

var CycleAnomalySpecs map[string]CycleAnomalySpecType
var CycleTypeNames map[string]struct{}

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
		RestRels:     restSet,
		FilterEx: filter,
	}
}

// TODO: implement this function.
func filterClosure(required string) FilterExType {
	return nil
}

func init()  {
	CycleAnomalySpecs = map[string]CycleAnomalySpecType{
		"G0": fromRelsWithFilter(core.WW),
		"G1c": fromFirstRelAndRest(core.WR, core.WW, core.WR),
		"G-single": fromFirstRelAndRest(core.RW, core.WW, core.WR),
		// TODO: G-nonadjacent is not implemented

		"G2-item": fromFirstRelAndRestWithFilter(filterClosure("G2-item"), core.RW, core.WR, core.RW, core.WW),
		"G0-process": fromRelsWithFilter(filterClosure("G0-process"), core.WW, core.Process),
		"G1c-process": fromFirstRelAndRestWithFilter(filterClosure("G1c-process"), core.RW, core.WW, core.WR, core.Process),
		"G-single-process": fromFirstRelAndRestWithFilter(filterClosure("G-single-process"), core.RW, core.WW, core.WR, core.Process),
		"G2-item-process": fromFirstRelAndRestWithFilter(filterClosure("G2-item-process"), core.RW, core.WW, core.WR, core.RW, core.Process),

		// realtime

		"G0-realtime": fromRelsWithFilter(filterClosure("G0-realtime"), core.WW, core.Realtime),
		"G1c-realtime": fromFirstRelAndRestWithFilter(filterClosure("G1c-realtime"), core.WR, core.WW, core.WR, core.Realtime),
		"G-single-realtime": fromFirstRelAndRestWithFilter(filterClosure("G-single-realtime"), core.RW, core.WW, core.WR, core.Realtime),
		"G2-item-realtime": fromFirstRelAndRestWithFilter(filterClosure("G2-item-realtime"), core.RW, core.WW, core.WR, core.Realtime, core.RW),
	}

	CycleTypeNames = map[string]struct{}{
		"G-nonadjacent-process": {},
		"G-nonadjacent-realtime": {},
	}

	for k, _ := range CycleAnomalySpecs {
		CycleTypeNames[k] = struct{}{}
	}
}

// CycleExplainWrapper is a ICycleExplainer, it's also a wrapper for core.
type CycleExplainerWrapper struct {}

func (c CycleExplainerWrapper) ExplainCycle(pairExplainer core.DataExplainer, circle core.Circle) (core.Circle, []core.Step) {
	panic("implement me")
}

func (c CycleExplainerWrapper) RenderCycleExplanation(explainer core.DataExplainer, circle core.Circle) string {
	return CycleExplainer{}.RenderCycleExplanation(explainer, circle)
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
	Anomalies []string

}

func AdditionalGraphs(opts interface{})  {
	panic("implement me")
}

func prohibitAnomalyTypes(cm []core.ConsistencyModelName, anomalies []string) map[core.Anomaly]struct{} {
	panic("implement me")
}

func compactAnomalies(anomalies ...core.Anomaly) map[core.Anomaly]struct{} {
	panic("implement me")
}
