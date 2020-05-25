package rwregister

import (
	"log"
	"strings"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

const initMagicNumber = -1919810

// previouslyWriteElement returns:
// * nil if mop is not an write mop
// * or initMagicNumber if mop write the initial value of that key
// * otherwise return the previous MopValueType value
// func previouslyWriteElement(writeIndex writeIdx, op core.Op, mop core.Mop) core.MopValueType {
// 	if !mop.IsWrite() {
// 		return nil
// 	}
// 	k := mop.GetKey()
// 	v := mop.GetValue().(int)
// 	appendSeq := appendIndex[k]
// 	index := -1
// 	for i := range appendSeq {
// 		if appendSeq[i] == v {
// 			index = i
// 			break
// 		}
// 	}
// 	// we don't found the value mop appended on appendIdx
// 	if index < 0 {
// 		return nil
// 	}
// 	if index > 0 {
// 		return appendIndex[k][index-1]
// 	}
// 	return initMagicNumber
// }

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
