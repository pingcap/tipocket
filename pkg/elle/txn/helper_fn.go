package txn

import (
	"hash/fnv"
	"sort"
	"strconv"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

type OpMopIterator struct {
	history      core.History
	historyIndex int
	mopIndex     int
}

func (omi *OpMopIterator) Next() (*core.Op, core.Mop) {
	op, mop := &omi.history[omi.historyIndex], (*omi.history[omi.historyIndex].Value)[omi.mopIndex]

	// inc index
	omi.mopIndex++
	if omi.mopIndex == omi.history[omi.historyIndex].ValueLength() {
		omi.mopIndex = 0
		omi.historyIndex++
	}
	return op, mop
}

func (omi *OpMopIterator) HasNext() bool {
	return omi.historyIndex < len(omi.history)
}

// OpMops return an iterator for history.
func OpMops(history core.History) *OpMopIterator {
	return &OpMopIterator{history: history, historyIndex: 0, mopIndex: 0}
}

// keeps or ok records and the records satisfied the validate fn.
func OkKeep(validateFunc func(op core.Op) bool, history core.History) core.History {
	var newHistory core.History
	for _, v := range history {
		if v.Type == core.OpTypeOk && validateFunc(v) {
			newHistory = append(newHistory, v)
		}
	}
	return newHistory
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

	for _, op := range history {
		final := map[string]core.MopValueType{}
		for _, mop := range *op.Value {
			if mop.IsAppend() {
				a := mop.(core.Append)
				realKey := a.Key
				lastOp, exists := final[realKey]
				if !exists {
					final[realKey] = a.Value
				} else {
					im[realKey][lastOp] = &op
				}
			}
		}
	}

	return im
}

// FailedWrites is like IntermediateWrites, it returns map[key](map[aborted-value]abort-op).
func FailedWrites(history core.History) map[string]map[core.MopValueType]*core.Op {
	im := map[string]map[core.MopValueType]*core.Op{}

	for _, op := range history {
		if op.Type != core.OpTypeFail {
			continue
		}
		for _, mop := range *op.Value {
			if mop.IsAppend() {
				a := mop.(core.Append)
				realKey := a.Key
				im[realKey][a.Value] = &op
			}
		}
	}

	return im
}

func mustAtoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return n
}

func ResultMap() {
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

func arrayHash(sset []core.Rel) uint32 {
	h := fnv.New32a()
	for _, v := range sset {
		h.Write([]byte(v))
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
		} else {
			g = graph.FilterRelationships(rels)
			if g != nil {
				memo[v] = g
			}
			return g
		}
	}
}
