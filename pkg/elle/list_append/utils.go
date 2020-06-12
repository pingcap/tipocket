package listappend

import (
	"fmt"
	"log"
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
func verifyUniqueAppends(history core.History) {
	tempDict := map[string]map[core.MopValueType]struct{}{}
	iter := txn.OpMops(history)
	for iter.HasNext() {
		op, mop := iter.Next()
		if op.Type == core.OpTypeInvoke && mop.IsAppend() {
			_, e := tempDict[mop.GetKey()]
			if !e {
				tempDict[mop.GetKey()] = map[core.MopValueType]struct{}{}
			}
			_, e = tempDict[mop.GetKey()][mop.GetValue()]
			if e {
				log.Panicf("duplicate appends, op %s, key: %s, value: %+v", op.String(), mop.GetKey(), mop.GetValue())
			}
			tempDict[mop.GetKey()][mop.GetValue()] = struct{}{}
		}
	}
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

type duplicateConflict struct {
	Op   core.Op
	Mop  core.Mop
	Dups map[core.MopValueType]int
}

func (d duplicateConflict) IAnomaly() {}

func (d duplicateConflict) String() string {
	return fmt.Sprintf("(DuplicateConflict) Op: %s, mop: %s, dups: %v", d.Op, d.Mop, d.Dups)
}

// duplicates does a "single-read-count", in :r [1 1 4], it will checks if
//  there is any duplicate words, and put it into anomalies structure.
func duplicates(history core.History) GCaseTp {
	iter := txn.OpMops(history)
	var duplicateCases []core.Anomaly
	for iter.HasNext() {
		op, mop := iter.Next()
		if mop.IsRead() {
			var reads []int
			if mop.GetValue() != nil {
				reads = mop.GetValue().([]int)
			}
			readCnt := map[core.MopValueType]int{}
			dups := map[core.MopValueType]int{}
			for _, v := range reads {
				readCnt[v] = readCnt[v] + 1
				if readCnt[v] > 1 {
					dups[v] = readCnt[v]
				}
			}
			if len(dups) > 0 {
				duplicateCases = append(duplicateCases, duplicateConflict{
					Op:   op,
					Mop:  mop,
					Dups: dups,
				})
			}
		}
	}
	return duplicateCases
}

func writeIndex(history core.History) writeIdx {
	indexMap := map[string]map[core.MopValueType]core.Op{}
	for _, op := range history {
		if op.Value == nil {
			continue
		}
		for _, mop := range *op.Value {
			if mop.IsAppend() {
				innerMap, e := indexMap[mop.GetKey()]
				if !e {
					indexMap[mop.GetKey()] = map[core.MopValueType]core.Op{}
					innerMap = indexMap[mop.GetKey()]
				}
				innerMap[mop.GetValue()] = op
			}
		}
	}
	return indexMap
}

func readIndex(history core.History) readIdx {
	indexRead := map[string]map[core.MopValueType][]core.Op{}
	for _, op := range history {
		if op.Value == nil {
			continue
		}
		for _, mop := range *op.Value {
			if !mop.IsRead() {
				continue
			}
			var k = mop.GetKey()
			var v []int
			if mop.GetValue() != nil {
				v = mop.GetValue().([]int)
			}
			// we jump not read mop
			// we jump info type and empty read
			if op.Type == core.OpTypeInfo && len(v) == 0 {
				continue
			}
			if _, ok := indexRead[k]; !ok {
				indexRead[k] = make(map[core.MopValueType][]core.Op)
			}
			var mopValue = core.MopValueType(initMagicNumber)
			if len(v) != 0 {
				mopValue = v[len(v)-1]
			}
			indexRead[k][mopValue] = append(indexRead[k][mopValue], op)
		}
	}
	return indexRead
}

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
func wrMopDep(writeIndex writeIdx, op core.Op, mop core.Mop) *core.Op {
	if mop.IsRead() {
		writeMap, e := writeIndex[mop.GetKey()]
		var v []int
		if mop.GetValue() != nil {
			v = mop.GetValue().([]int)
		}
		if len(v) == 0 {
			return nil
		}
		lastRead := v[len(v)-1]
		appendOp, e := writeMap[core.MopValueType(lastRead)]
		if !e {
			return nil
		}
		// Note: I don't know where comparing like this is ok.
		if op != appendOp {
			return &appendOp
		}
	}
	return nil
}

// previouslyAppendElement returns:
// * nil if mop is not a append mop
// * or initMagicNumber if mop append the first element of that key
// * otherwise return the previous MopValueType value
func previouslyAppendElement(appendIndex appendIdx, op core.Op, mop core.Mop) core.MopValueType {
	if !mop.IsAppend() {
		return nil
	}
	k := mop.GetKey()
	v := mop.GetValue().(int)
	appendSeq := appendIndex[k]
	index := -1
	for i := range appendSeq {
		if appendSeq[i] == v {
			index = i
			break
		}
	}
	// we don't found the value mop appended on appendIdx
	if index < 0 {
		return nil
	}
	if index > 0 {
		return appendIndex[k][index-1]
	}
	return initMagicNumber
}

// returns What (other) operation wrote the value just before this write mop?"
func wwMopDep(appendIndexResult appendIdx, writeIndexResult writeIdx, op core.Op, mop core.Mop) *core.Op {
	previousElement := previouslyAppendElement(appendIndexResult, op, mop)
	if previousElement == nil || previousElement == initMagicNumber {
		return nil
	}
	writerMap, e := writeIndexResult[mop.GetKey()]
	// It must exists, otherwise the algorithm of writeIndex is error.
	if !e {
		return nil
	}
	writer, e := writerMap[previousElement]
	if !e {
		return nil
	}
	if writer == op {
		return nil
	}
	return &writer
}

func rwMopDeps(appendIndexResult appendIdx, writeIndexResult writeIdx, readIndexResult readIdx, op core.Op, mop core.Mop) map[core.Op]struct{} {
	previousElement := previouslyAppendElement(appendIndexResult, op, mop)
	if previousElement == nil {
		return nil
	}
	mopDeps := map[core.Op]struct{}{}
	ops := readIndexResult[mop.GetKey()]
	for _, o := range ops[previousElement] {
		// filter out op
		if o == op {
			continue
		}
		mopDeps[o] = struct{}{}
	}
	return mopDeps
}

func mopDeps(appendIndexResult appendIdx, writeIndexResult writeIdx, readIndexResult readIdx, op core.Op, mop core.Mop) map[core.Op]struct{} {
	res := map[core.Op]struct{}{}
	if mop.IsAppend() {
		r := rwMopDeps(appendIndexResult, writeIndexResult, readIndexResult, op, mop)
		resDep := wwMopDep(appendIndexResult, writeIndexResult, op, mop)
		if resDep != nil {
			r[*resDep] = struct{}{}
		}
		res = r
	} else if mop.IsRead() {
		resDep := wrMopDep(writeIndexResult, op, mop)
		if resDep != nil {
			res[*resDep] = struct{}{}
		}
	} else {
		panic("unreachable")
	}
	return res
}

func opDeps(appendIndexResult appendIdx, writeIndexResult writeIdx, readIndexResult readIdx, op core.Op) map[core.Op]struct{} {
	response := map[core.Op]struct{}{}
	if op.Value == nil {
		return response
	}
	for k := range *op.Value {
		response = setUnion(response, mopDeps(appendIndexResult, writeIndexResult, readIndexResult, op, (*op.Value)[k]))
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
	}
	return mergeOrders(prefix, seq1, seq2[1:])
}

func valuesFromSingleAppend(history core.History) map[string]core.MopValueType {
	temp := map[string][]core.MopValueType{}
	iter := txn.OpMops(history)
	for iter.HasNext() {
		_, mop := iter.Next()
		if mop.IsAppend() {
			k, _ := temp[mop.GetKey()]
			// We can append, but for economize the memory, at most 2 value will be record.
			if len(k) < 2 {
				temp[mop.GetKey()] = append(k, mop.GetValue())
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
	// book whether a []core.MopValueExist is already in keyMaps[key]
	valueExistMap := map[string]map[uint32]struct{}{}
	for iter.HasNext() {
		_, mop := iter.Next()
		if !mop.IsRead() {
			continue
		}
		if mop.GetValue() != nil {
			values := mop.GetValue().([]int)
			// we ignore the initial state
			if len(values) == 0 {
				continue
			}
			hash := txn.IntArrayHash(values)
			if _, ok := valueExistMap[mop.GetKey()]; !ok {
				valueExistMap[mop.GetKey()] = make(map[uint32]struct{})
			}
			// the read sequence already on keyMaps[key] set, so we jump it
			if _, ok := valueExistMap[mop.GetKey()][hash]; ok {
				continue
			}
			valueWithRightTp := make([]core.MopValueType, len(values), len(values))
			for i, v := range values {
				valueWithRightTp[i] = core.MopValueType(v)
			}
			keyMaps[mop.GetKey()] = append(keyMaps[mop.GetKey()], valueWithRightTp)
			valueExistMap[mop.GetKey()][hash] = struct{}{}
		}
	}

	singleAppends := valuesFromSingleAppend(history)

	for k, v := range singleAppends {
		if _, ok := keyMaps[k]; !ok {
			keyMaps[k] = [][]core.MopValueType{{v}}
		}
	}

	for k, v := range keyMaps {
		sort.Sort(core.SameKeyOpsByLength(v))
		keyMaps[k] = v
	}

	return keyMaps
}

type incompatibleOrder struct {
	Key    string
	Values [][]core.MopValueType
}

func (i incompatibleOrder) IAnomaly() {}

// Note: maybe i should format the values.
func (i incompatibleOrder) String() string {
	return fmt.Sprintf("(IncompatibleOrder) Key: %s, values: %v", i.Key, i.Values)
}

// incompatibleOrders return anomaly maps.
// return
// * nil or length of anomalies is zero: if no anomalies (so please use len() to check).
// * map[key](anomaly one, anomaly two)
func incompatibleOrders(sortedValues map[string][][]core.MopValueType) []core.Anomaly {
	var anomalies []core.Anomaly
	for k, values := range sortedValues {
		for index := 1; index < len(values); index++ {
			a, b := values[index-1], values[index]
			if seqIsPrefix(a, b) == false {
				anomalies = append(anomalies, incompatibleOrder{Key: k, Values: [][]core.MopValueType{a, b}})
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

func preProcessHistory(history core.History) core.History {
	history = core.FilterOutNemesisHistory(history)
	history.AttachIndexIfNoExists()
	return history
}

func isReadRecordEqual(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for idx := range a {
		if a[idx] != b[idx] {
			return false
		}
	}
	return true
}
