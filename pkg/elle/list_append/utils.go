package list_append

import (
	"sort"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

// Note: it will check if all mops are :append and :r
//  In our current implement, it's true.
func verifyMopTypes(mop core.Mop) bool {
	return true
}

// Takes a history of txns made up of appends and reads, and checks to make
//  sure that every invoke appending a value to a key chose a unique value.
func verifyUniqueAppends(history core.History) bool {
	panic("implement me")
}

// Takes a map of keys to observed values (e.g. from
//  `sorted-values`, and verifies that for each key, the values
//  read are consistent with a total order of appends. For instance, these values
//  are consistent:
// Returns nil if the history is OK, or throws otherwise.
func verifyTotalOrders(history core.History) bool {
	sorted := sortedValues(history)
	return len(incompatibleOrders(sorted)) == 0
}

// As a special case of sorted-values, if we have a key which only has a single
//  append, we don't need a read: we can infer the sorted values it took on were
//  simply [], [x].
func seqIsPrefix(seqBefore, seqAfter []core.MopValueType) bool {
	if len(seqBefore) > len(seqAfter) {
		return false
	}
	minLength := len(seqBefore)
	for i := 0; i < minLength; i++ {
		if seqBefore[i] != seqAfter[i] {
			return false
		}
	}
	return true
}

// duplicates does a "single-read-count", in :r [1 1 4], it will checks if
//  there is any duplicate words, and put it into anomalies structure.
// Note: The return of the function may not match the original.
func duplicates(history core.History) map[string][][]core.MopValueType {
	iter := txn.OpMops(history)
	anomalies := map[string][][]core.MopValueType{}
	for iter.HasNext() {
		_, mop := iter.Next()
		if mop == nil || *mop == nil {
			continue
		}
		if (*mop).IsRead() {
			if (*mop).GetValue() == nil {
				continue
			}
			reads := (*mop).GetValue().([]int)
			mopReads := make([]core.MopValueType, len(reads), len(reads))
			for i, v := range reads {
				mopReads[i] = v
			}
			readCnt := map[core.MopValueType]int{}
			for _, v := range reads {
				readCnt[v] = readCnt[v] + 1
				if readCnt[v] == 2 {
					anomalies[(*mop).GetKey()] = append(anomalies[(*mop).GetKey()], mopReads)
				}
			}
		}
	}
	return anomalies
}

type writeIdx = map[string]map[core.MopValueType]core.Op

func writeIndex(history core.History) writeIdx {
	indexMap := map[string]map[core.MopValueType]core.Op{}
	for _, op := range history {
		if op.Value == nil {
			continue
		}
		for _, mop := range *op.Value {
			if mop == nil || *mop == nil {
				continue
			}
			if mop.IsAppend() {
				innerMap, e := indexMap[mop.GetKey()]
				if !e {
					indexMap[mop.GetKey()] = map[core.MopValueType]core.Op{}
					innerMap = indexMap[mop.GetKey()]
				}
				innerMap[mop.GetKey()] = op
			}
		}
	}
	return indexMap
}

type readIdx = map[string][]core.Op

func readIndex(history core.History) readIdx {
	indexRead := map[string][]core.Op{}
	for _, op := range history {
		if op.Value == nil {
			continue
		}
		for _, mop := range *op.Value {
			if mop == nil || *mop == nil {
				continue
			}
			if mop.IsRead() && mop.GetValue() != nil {
				indexRead[mop.GetKey()] = append(indexRead[mop.GetKey()], op)
			}
		}
	}
	return indexRead
}

type appendIdx = map[string][]core.MopValueType

func appendIndex(sortedValues map[string][][]core.MopValueType) appendIdx {
	res := map[string][]core.MopValueType{}
	for k, v := range sortedValues {
		var current []core.MopValueType
		for index := 0; index < len(v); index++ {
			current = mergeOrders(nil, current, v[index])
		}
		res[k] = current
	}
	return res
}

// Note: mop should be a read, and should be part of op.
func wrMopDep(writeIndex writeIdx, op core.Op, mop *core.Mop) *core.Op {
	if mop != nil && (*mop).IsRead() {
		writeMap, e := writeIndex[(*mop).GetKey()]
		// Note: maybe unsafe
		if (*mop).GetValue() == nil {
			return nil
		}
		// (*mop).GetValue(), so it at least contains one value, it's OK to do like this
		firstValue := (*mop).GetValue().([]int)[0]
		op2, e := writeMap[core.MopValueType(firstValue)]
		if !e {
			return nil
		}
		// Note: I don't know where comparing like this is ok.
		if op == op2 {
			return &op
		}
	}
	return nil
}

// Note: return nil if
// * the input not match the requirements.
// The returning bool means "it's the initialize value", if it's true, we may return (nil, true).
// Should we have more return types here?
func previouslyAppendElement(appendIndexResult appendIdx, op core.Op, mop *core.Mop) (core.MopValueType, bool) {
	if mop == nil || !(*mop).IsAppend() {
		return nil, false
	}
	subdict, e := appendIndexResult[(*mop).GetKey()]
	if !e {
		return nil, false
	}
	// Note: It's an append, so append value should not be nil.
	appendValue := (*mop).GetValue()
	for i, v := range subdict {
		if v == appendValue {
			if i != 0 {
				return subdict[i-1], false
			} else {
				return nil, true
			}
		}
	}
	return nil, false
}

// returns What (other) operation wrote the value just before this write mop?"
func wwMopDep(appendIndexResult appendIdx, writeIndexResult writeIdx, op core.Op, mop *core.Mop) *core.Op {
	previousElement, isInit := previouslyAppendElement(appendIndexResult, op, mop)
	if isInit {
		// no writer precedes us
		return nil
	}
	if previousElement == nil {
		return nil
	}
	writerMap, e := writeIndexResult[(*mop).GetKey()]
	// It must exists, otherwise the algorithm of writeIndex is error.
	if !e {
		return nil
	}
	element, e := writerMap[previousElement]
	if !e {
		return nil
	}
	if element == op {
		return nil
	}
	return &element
}

func rwMopDeps(appendIndexResult appendIdx, writeIndexResult writeIdx, readIndexResult readIdx, op core.Op, mop *core.Mop) map[core.Op]struct{} {
	previousElement, isInit := previouslyAppendElement(appendIndexResult, op, mop)
	if isInit {
		// no writer precedes us
		return nil
	}
	if previousElement == nil {
		return nil
	}
	mopDeps := map[core.Op]struct{}{}
	// Note: if previousElement is not nil, mop must can call `GetKey`.
	ops, e := readIndexResult[(*mop).GetKey()]
	if !e {
		return nil
	}
	for _, v := range ops {
		if v != previousElement {
			mopDeps[v] = struct{}{}
		}
	}
	return mopDeps
}

func mopDeps(appendIndexResult appendIdx, writeIndexResult writeIdx, readIndexResult readIdx, op core.Op, mop *core.Mop) map[core.Op]struct{} {
	if mop == nil {
		return nil
	}
	res := map[core.Op]struct{}{}
	if (*mop).IsAppend() {
		r := rwMopDeps(appendIndexResult, writeIndexResult, readIndexResult, op, mop)
		resDep := wwMopDep(appendIndexResult, writeIndexResult, op, mop)
		if resDep != nil {
			r[*resDep] = struct{}{}
		}
		res = r
	} else {
		resDep := wrMopDep(writeIndexResult, op, mop)
		if resDep != nil {
			res[*resDep] = struct{}{}
		}
	}
	return res
}

func opDeps(appendIndexResult appendIdx, writeIndexResult writeIdx, readIndexResult readIdx, op core.Op) map[core.Op]struct{} {
	response := map[core.Op]struct{}{}
	if op.Value == nil {
		return response
	}
	for k, _ := range *op.Value {
		response = setUnion(response, mopDeps(appendIndexResult, writeIndexResult, readIndexResult, op, &(*op.Value)[k]))
	}
	return response
}

func setUnion(dest, src2 map[core.Op]struct{}) map[core.Op]struct{} {
	if src2 == nil {
		return dest
	}

	for k := range src2 {
		dest[k] = struct{}{}
	}
	return dest
}

func mergeOrders(prefix, seq1, seq2 []core.MopValueType) []core.MopValueType {
	if len(seq1) == 0 {
		return append(prefix, seq2...)
	}
	if len(seq2) == 0 {
		return append(prefix, seq1...)
	}
	if seq1[0] == seq2[0] {
		return mergeOrders(append(prefix, seq1[0]), seq1[1:], seq2[1:])
	}
	i1 := seq1[0].(int)
	i2 := seq2[0].(int)
	if i1 < i2 {
		return mergeOrders(prefix, seq1[1:], seq2)
	} else {
		return mergeOrders(prefix, seq1, seq2[1:])
	}
}

func valuesFromSingleAppend(history core.History) map[string]core.MopValueType {
	temp := map[string][]core.MopValueType{}
	iter := txn.OpMops(history)
	for iter.HasNext() {
		_, mop := iter.Next()
		if (*mop).IsAppend() {
			k, _ := temp[(*mop).GetKey()]
			// We can append, but for economize the memory, at most 2 value will be record.
			if len(k) < 2 {
				temp[(*mop).GetKey()] = append(k, (*mop).GetValue())
			}
		}
	}
	result := map[string]core.MopValueType{}
	for k, v := range temp {
		if len(v) == 1 {
			result[k] = v[0]
		}
	}
	return result
}

// Takes a history where operation values are transactions, and every micro-op
//  in a transaction is an append or a read. Computes a map of keys to all
//  distinct observed values for that key, ordered by length.
//
// Note: This helper function is important, it's return represents map[key][](The order this key should read).
func sortedValues(history core.History) map[string][][]core.MopValueType {
	iter := txn.OpMops(history)
	keyMaps := map[string][][]core.MopValueType{}
	for iter.HasNext() {
		_, mop := iter.Next()
		if mop == nil || *mop == nil {
			continue
		}
		if !(*mop).IsRead() {
			continue
		}
		if (*mop).GetValue() != nil {
			values := (*mop).GetValue().([]core.MopValueType)
			keyMaps[(*mop).GetKey()] = append(keyMaps[(*mop).GetKey()], values)
		}
	}

	singleAppends := valuesFromSingleAppend(history)

	for k, _ := range singleAppends {
		a, e := singleAppends[k]
		if !e {
			keyMaps[k] = [][]core.MopValueType{{a}}
		}
	}

	for k, v := range keyMaps {
		sort.Sort(core.SameKeyOpsByLength(v))
		keyMaps[k] = v
	}

	return keyMaps
}

// incompatibleOrders return anomaly maps.
// return
// * nil or length of anomalies is zero: if no anomalies (so please use len() to check).
// * map[key](anomaly one, anomaly two)
func incompatibleOrders(sortedValues map[string][][]core.MopValueType) map[string][]core.MopValueType {
	anomalies := map[string][]core.MopValueType{}

	for k, v := range sortedValues {
		for index := 1; index < len(v); index++ {
			if seqIsPrefix(v[index-1], v[index]) {
				anomalies[k] = []core.MopValueType{
					v[index-1], v[index],
				}
			}
		}
	}

	return anomalies
}

func min(a, b int) int {
	if a > b {
		return b
	}
	return a
}
