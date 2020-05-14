package txn

import (
	"encoding/binary"
	"hash/fnv"
	"sort"
	"strconv"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

// OpMopIterator builds a op-mop iterator
type OpMopIterator struct {
	history      core.History
	historyIndex int
	mopIndex     int
}

// Next iterates next op-mop
func (omi *OpMopIterator) Next() (core.Op, core.Mop) {
	op, mop := omi.history[omi.historyIndex], (*omi.history[omi.historyIndex].Value)[omi.mopIndex]

	// inc index
	omi.mopIndex++
	if omi.mopIndex == omi.history[omi.historyIndex].ValueLength() {
		omi.mopIndex = 0
		omi.historyIndex++
	}
	return op, mop
}

// HasNext returns whether the iterator has ended
func (omi *OpMopIterator) HasNext() bool {
	return omi.historyIndex < len(omi.history)
}

// OpMops return an iterator for history.
func OpMops(history core.History) *OpMopIterator {
	return &OpMopIterator{history: history, historyIndex: 0, mopIndex: 0}
}

// Gen Takes a sequence of transactions and returns a sequence of invocation operations.
// Note: Currently we don't need :f.
func Gen(mop []core.Mop) core.Op {
	return core.Op{
		Type:  core.OpTypeInvoke,
		Value: &mop,
	}
}

// IntermediateWrites return a map likes map[key](map[old-version]overwrite-op).
// Note: This function is very very strange, please pay attention to it carefully.
func IntermediateWrites(history core.History) map[string]map[core.MopValueType]*core.Op {
	im := map[string]map[core.MopValueType]*core.Op{}

	for idx, op := range history {
		if op.Value == nil {
			continue
		}
		final := map[string]core.MopValueType{}
		for _, mop := range *op.Value {
			if mop.IsAppend() {
				realKey := mop.GetKey()
				if lastOp, exists := final[realKey]; exists {
					if _, ok := im[realKey]; !ok {
						im[realKey] = make(map[core.MopValueType]*core.Op)
					}
					im[realKey][lastOp] = &history[idx]
				}
				final[realKey] = mop.GetValue()
			}
		}
	}

	return im
}

// FailedWrites is like IntermediateWrites, it returns map[key](map[aborted-value]abort-op).
func FailedWrites(history core.History) map[string]map[core.MopValueType]*core.Op {
	failed := map[string]map[core.MopValueType]*core.Op{}

	for idx := range history {
		op := &history[idx]
		if op.Type != core.OpTypeFail {
			continue
		}
		for _, mop := range *op.Value {
			if mop.IsAppend() {
				realKey := mop.GetKey()
				if _, ok := failed[realKey]; !ok {
					failed[realKey] = make(map[core.MopValueType]*core.Op)
				}
				failed[realKey][mop.GetValue()] = op
			}
		}
	}
	return failed
}

func mustAtoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return n
}

// ResultMap takes opts and processes anomalies
func ResultMap(opts Opts, anomalies core.Anomalies) CheckResult {
	bad := anomalies.SelectKeys(prohibitedAnomalyTypes(opts.ConsistencyModels, opts.Anomalies))
	reportable := anomalies.SelectKeys(reportableAnomalyTypes(opts.ConsistencyModels, opts.Anomalies))

	if len(reportable) == 0 {
		return CheckResult{
			Valid: true,
		}
	}
	cr := CheckResult{}
	if len(bad) != 0 {
		cr.Valid = false
	} else if len(reportable) != 0 {
		cr.IsUnknown = true
	} else {
		cr.Valid = true
	}
	cr.AnomalyTypes = reportable.Keys()
	cr.Anomalies = reportable
	cr.Not, cr.AlsoNot = core.FriendlyBoundary(anomalies.Keys())
	return cr
}

// Note: maybe cache this function is better?
func setKeys(m map[core.Rel]struct{}) []core.Rel {
	var rels []core.Rel
	for k := range m {
		rels = append(rels, k)
	}
	return rels
}

func arrayHash(sset []core.Rel) uint32 {
	h := fnv.New32a()
	for _, v := range sset {
		h.Write([]byte(v))
	}
	return h.Sum32()
}

// IntArrayHash maps []int to uint32
func IntArrayHash(array []int) uint32 {
	h := fnv.New32a()
	for _, v := range array {
		bs := make([]byte, 8)
		binary.LittleEndian.PutUint64(bs, uint64(v))
		h.Write(bs)
	}
	return h.Sum32()
}

// FilteredGraphs receives a graph and a collection of relations, return a new Graph filtered to just those relationships
// Note: currently it use fork here, we can considering remove it.
func FilteredGraphs(graph core.DirectedGraph) FilterGraphFn {
	memo := map[uint32]*core.DirectedGraph{}

	return func(rels []core.Rel) *core.DirectedGraph {
		sort.Sort(core.RelSet(rels))
		v := arrayHash(rels)
		if g, e := memo[v]; e {
			return g
		}
		g := graph.FilterRelationships(rels)
		if g != nil {
			memo[v] = g
		}
		return g
	}
}
