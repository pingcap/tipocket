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
}