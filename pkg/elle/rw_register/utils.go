package rwregister

import (
	"log"
	"strings"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

const initMagicNumber = -1919810

// Takes a history of txns made up of appends and reads, and checks to make
//  sure that every invoke writes a value to a key chose a unique value.
func verifyUniqueWrites(history core.History) {
	tempDict := map[string]map[int]struct{}{}
	iter := txn.OpMops(history)
	for iter.HasNext() {
		op, mop := iter.Next()
		if op.Type == core.OpTypeInvoke && mop.IsWrite() {
			for k, v := range mop.M {
				if _, e := tempDict[k]; !e {
					tempDict[k] = map[int]struct{}{}
				}
				if v == nil {
					panic("write value should not be nil")
				}
				vptr := v.(*int)
				if _, e := tempDict[k][*vptr]; e {
					log.Panicf("duplicate writes, op %s, key: %s, value: %+v", op.String(), k, *vptr)
				}
				tempDict[k][*vptr] = struct{}{}
			}
		}
	}
}

func writeIndex(history core.History) writeIdx {
	writeIdx := make(map[string]map[int]core.Op)

	for _, op := range history {
		for _, mop := range *op.Value {
			if !mop.IsWrite() {
				continue
			}
			for k, v := range mop.M {
				vptr := v.(*int)
				if vptr == nil {
					panic("write value should not be nil")
				}
				if _, ok := writeIdx[k]; !ok {
					writeIdx[k] = make(map[int]core.Op)
				}
				writeIdx[k][*vptr] = op
			}
		}
	}

	return writeIdx
}

func readIndex(history core.History) readIdx {
	readIdx := make(map[string]map[int][]core.Op)

	for _, op := range history {
		for _, mop := range *op.Value {
			if !mop.IsRead() {
				continue
			}
			for k, v := range mop.M {
				vptr := v.(*int)
				if vptr == nil {
					panic("write value should not be nil")
				}
				if _, ok := readIdx[k]; !ok {
					readIdx[k] = make(map[int][]core.Op)
				}
				if _, ok := readIdx[k][*vptr]; !ok {
					readIdx[k][*vptr] = make([]core.Op, 0)
				}
				readIdx[k][*vptr] = append(readIdx[k][*vptr], op)
			}
		}
	}

	return readIdx
}

func preprocess(h core.History) (history core.History, writeIdx writeIdx, readIdx readIdx) {
	verifyUniqueWrites(h)

	history = core.FilterOkOrInfoHistory(h)

	writeIdx = writeIndex(history)
	readIdx = readIndex(history)
	return
}

// func previouslyWriteOp(writeIndexResult writeIdx, op core.Op, mop core.Mop) *core.Op {

// }

func wrMopDep(writeIndexResult writeIdx, op core.Op, mop core.Mop) map[core.Op]struct{} {
	mopDeps := map[core.Op]struct{}{}
	for k, v := range mop.M {
		writeMap, ok := writeIndexResult[k]
		if !ok {
			continue
		}
		vptr := v.(*int)
		if vptr == nil {
			panic("write value should not be nil")
		}
		o, ok := writeMap[*vptr]
		if !ok {
			// this is an abort read which should violate g1a
			// skip it here
			continue
		}
		if o == op {
			continue
		}
		mopDeps[o] = struct{}{}
	}
	return mopDeps
}

func extKeys(op core.Op) map[string]struct{} {
	keys := make(map[string]struct{})
	for _, mop := range *op.Value {
		if !mop.IsRead() && !mop.IsWrite() {
			continue
		}
		if mop.IsRead() {
			if op.Type != core.OpTypeOk {
				continue
			}
		}
		if mop.IsWrite() {
			if op.Type != core.OpTypeOk && op.Type != core.OpTypeInfo {
				continue
			}
		}
		for k := range mop.M {
			keys[k] = struct{}{}
		}
	}
	return keys
}

func extReadKeys(op core.Op) map[string]int {
	var (
		res    = make(map[string]int)
		ignore = make(map[string]int)
	)

	for _, mop := range *op.Value {
		for k, v := range mop.M {
			vptr := v.(*int)
			if vptr == nil {
				continue
			}
			_, ok := ignore[k]
			if !ok && mop.IsRead() {
				res[k] = *vptr
			}
			ignore[k] = *vptr
		}
	}

	return res
}

func extWriteKeys(op core.Op) map[string]int {
	var res = make(map[string]int)

	for _, mop := range *op.Value {
		for k, v := range mop.M {
			vptr := v.(*int)
			if vptr == nil {
				continue
			}
			if mop.IsWrite() {
				res[k] = *vptr
			}
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

type pairData struct {
	k         string
	prevValue int
	value     int
}

func explainPairData(a, b core.PathType, aTp, bTp core.MopType) *pairData {
	for _, bmop := range *b.Value {
		if bmop.T != bTp {
			continue
		}
		for _, amop := range *b.Value {
			if amop.T != aTp {
				continue
			}
			for bk, bval := range bmop.M {
				bvptr := bval.(*int)
				if bvptr == nil {
					continue
				}
				for ak, aval := range amop.M {
					if ak != bk {
						continue
					}
					avptr := aval.(*int)
					if avptr == nil {
						continue
					}
					if *avptr == *bvptr {
						return &pairData{
							k:         bk,
							prevValue: *avptr,
							value:     *bvptr,
						}
					}
				}
			}
		}
	}
	return nil
}

// explainOpDeps
// Given version graphs, a function extracting a map of keys to values from op
// A, and also from op B, and a pair of operations A and B, returns a map (or
// nil) explaining why A precedes B.
func explainOpDeps(graphs map[string]*core.DirectedGraph,
	extA func(op core.Op) map[string]int, a core.Op,
	extB func(op core.Op) map[string]int, b core.Op) (string, int, int) {
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
			if out.Value.(int) == bval {
				return k, aval, bval
			}
		}
	}
	return "", initMagicNumber, initMagicNumber
}
