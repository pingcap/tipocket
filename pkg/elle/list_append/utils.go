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
		if (*mop).IsRead() {
			reads := (*mop).GetValue().([]core.MopValueType)
			readCnt := map[core.MopValueType]int{}
			for _, v := range reads {
				readCnt[v] = readCnt[v] + 1
				if readCnt[v] == 2 {
					anomalies[(*mop).GetKey()] = append(anomalies[(*mop).GetKey()], reads)
				}
			}
		}
	}
	return anomalies
}

func writeIndex(history core.History) WriteIdx {
	panic("implement me")
}

func readIndex(history core.History) ReadIdx {
	panic("implement me")
}

func appendIndex(sortedValues map[string][][]core.MopValueType) AppendIdx {
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
		if !(*mop).IsRead() {
			continue
		}
		values := (*mop).GetValue().([]core.MopValueType)
		keyMaps[(*mop).GetKey()] = append(keyMaps[(*mop).GetKey()], values)
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

// Takes a history of operations, where each operation op has a :value which is
//  a transaction made up of [f k v] micro-ops. Runs a reduction over every
//  micro-op, where the reduction function is of the form (f state op [f k v]).
//  Saves you having to do endless nested reduces.
func reduceMops() {

}

func wwMopDep(appendIdx AppendIdx, writeIdx WriteIdx, op core.Op, mop core.Mop) *core.Op {
	panic("impl")
}

func wrMopDep(writeIdx WriteIdx, op core.Op, mop core.Mop) *core.Op {
	panic("impl")
}

func previousAppendedElement(appendIdx AppendIdx, writeIdx WriteIdx, op core.Op, mop core.Mop) core.MopValueType {
	panic("impl")
}
