package txn

import (
	"strconv"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

type OpMopIterator struct {
	history      core.History
	historyIndex int
	mopIndex     int
}

func (omi *OpMopIterator) Next() (*core.Op, core.Mop) {
	op, mop := &omi.history[omi.historyIndex], omi.history[omi.historyIndex].Value[omi.mopIndex]

	// inc index
	if omi.mopIndex == len(omi.history[omi.historyIndex].Value) {
		omi.mopIndex = 0
		omi.historyIndex++
	} else {
		omi.mopIndex++
	}
	return op, mop
}

func (omi *OpMopIterator) HasNext() bool {
	return omi.historyIndex < len(omi.history)
}

// OpMops return an iterator for history.
func OpMops(history core.History) *OpMopIterator {
	return &OpMopIterator{history: history, historyIndex: 0, mopIndex: 0, op: nil}
}

// keeps or ok records and the records satisfied the validate fn.
func OkKeep(validateFunc func(op core.Op) bool, history core.History) core.History {
	var newHistory core.History
	for _, v := range history {
		if v.Type == core.Ok && validateFunc(v) {
			newHistory = append(newHistory, v)
		}
	}
	return newHistory
}

// Gen Takes a sequence of transactions and returns a sequence of invocation operations.
// TODO: make clear if we need `:f`.
func Gen(mop []core.Mop) core.Op {
	return core.Op{
		Type:  core.Invoke,
		Value: mop,
	}
}

// IntermediateWrites return a map likes map[key](map[old-version]overwrite-op).
// Note: This function is very very strange, please pay attention to it carefully.
func IntermediateWrites(history core.History) map[int]map[core.MopValueType]*core.Op {
	final := map[int]core.MopValueType{}
	im := map[int]map[core.MopValueType]*core.Op{}

	for _, op := range history {
		for _, mop := range op.Value {
			if mop.IsAppend() {
				a := mop.(core.Append)
				realKey := mustAtoi(a.Key)
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

func FailedWrites(history core.History) {
	panic("implement me")
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
