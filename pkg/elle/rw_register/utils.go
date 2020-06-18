package rwregister

import (
	"strings"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

func extReadKeys(op core.Op) map[string]Int {
	var (
		res    = make(map[string]Int)
		ignore = make(map[string]struct{})
	)

	for _, mop := range *op.Value {
		k, v := mop.GetKey(), mop.GetValue()
		i := v.(Int)
		_, ok := ignore[k]
		if !ok && mop.IsRead() {
			res[k] = i
		}
		ignore[k] = struct{}{}
	}

	return res
}

func extWriteKeys(op core.Op) map[string]Int {
	var res = make(map[string]Int)

	for _, mop := range *op.Value {
		k, v := mop.GetKey(), mop.GetValue()
		if mop.IsWrite() {
			i := v.(Int)
			if i.IsNil {
				panic("should not write with nil")
			}
			res[k] = i
		}
	}

	return res
}

func isExtIndexRel(rel core.Rel) (string, bool) {
	if strings.HasPrefix(string(rel), string(core.ExtKey)) && len(rel) > 8 {
		return string(rel[8:]), true
	}
	return "", false
}

func preProcessHistory(history core.History) core.History {
	history = core.FilterOutNemesisHistory(history)
	history.AttachIndexIfNoExists()
	return history
}

// explainOpDeps
// Given version graphs, a function extracting a map of keys to values from op
// A, and also from op B, and a pair of operations A and B, returns a map (or
// nil) explaining why A precedes B.
func explainOpDeps(graphs map[string]*core.DirectedGraph,
	extA func(op core.Op) map[string]Int, a core.Op,
	extB func(op core.Op) map[string]Int, b core.Op) (string, Int, Int) {
	var (
		akvs = extA(a)
		bkvs = extB(b)
	)
	for k, aval := range akvs {
		graph, ok := graphs[k]
		if !ok {
			continue
		}
		bval, ok := bkvs[k]
		if !ok {
			continue
		}
		outs, ok := graph.Outs[core.Vertex{Value: aval}]
		if !ok {
			continue
		}
		for out := range outs {
			if out.Value.(Int).Eq(bval) {
				return k, aval, bval
			}
		}
	}
	return "", NewNil(), NewNil()
}
