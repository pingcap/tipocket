package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const PaparExample = `{:index 0 :type :invoke  :value [[:append 253 1] [:append 253 3] [:append 253 4] [:append 255 2] [:append 255 3] [:append 255 4] [:append 255 5] [:append 256 1] [:append 256 2]]}
{:index 1 :type :ok      :process 1 :value [[:append 253 1] [:append 253 3] [:append 253 4] [:append 255 2] [:append 255 3] [:append 255 4] [:append 255 5] [:append 256 1] [:append 256 2]]}
{:index 2 :type :invoke, :process 1 :value [[:append 255 8] [:r 253 nil]]}
{:index 3 :type :ok,     :value [[:append 255 8] [:r 253 [1 3 4]]]}
{:index 4 :type :invoke, :process 2 :value [[:append 256 4] [:r 255 nil] [:r 256 nil] [:r 253 nil]]}
{:index 5 :type :ok,     :value [[:append 256 4] [:r 255 [2 3 4 5 8]] [:r 256 [1 2 4]] [:r 253 [1 3 4]]]}
{:index 6 :type :invoke, :value [[:append 250 10] [:r 253 nil] [:r 255 nil] [:append 256 3]]}
{:index 7 :type :ok      :value [[:append 250 10] [:r 253 [1 3 4]] [:r 255 [2 3 4 5]] [:append 256 3]]}`

// copy i and return its pointer
func int2pointer(i int) *int {
	return &i
}

func TestParseHistory(t *testing.T) {
	txn1Mops := []Mop{
		Append{Key: "253", Value: 1},
		Append{Key: "253", Value: 3},
		Append{Key: "253", Value: 4},
		Append{Key: "255", Value: 2},
		Append{Key: "255", Value: 3},
		Append{Key: "255", Value: 4},
		Append{Key: "255", Value: 5},
		Append{Key: "256", Value: 1},
		Append{Key: "256", Value: 2},
	}
	txn3MopsInvoke := []Mop{
		Append{Key: "256", Value: 4},
		Read{Key: "255", Value: nil},
		Read{Key: "256", Value: nil},
		Read{Key: "253", Value: nil},
	}
	txn3MopsOk := []Mop{
		Append{Key: "256", Value: 4},
		Read{Key: "255", Value: []int{2, 3, 4, 5, 8}},
		Read{Key: "256", Value: []int{1, 2, 4}},
		Read{Key: "253", Value: []int{1, 3, 4}},
	}
	txn5MopsInvoke := []Mop{
		Append{Key: "250", Value: 10},
		Read{Key: "253", Value: nil},
		Read{Key: "255", Value: nil},
		Append{Key: "256", Value: 3},
	}
	txn5MopsOk := []Mop{
		Append{Key: "250", Value: 10},
		Read{Key: "253", Value: []int{1, 3, 4}},
		Read{Key: "255", Value: []int{2, 3, 4, 5}},
		Append{Key: "256", Value: 3},
	}

	history, err := ParseHistory(PaparExample)
	assert.Equal(t, err, nil, "parse history, no error")
	assert.Equal(t, len(history), 8, "parse history, length")
	assert.Equal(t, history[0], Op{Index: 0, Type: OpTypeInvoke, Value: txn1Mops}, "parse history, history[0]")
	assert.Equal(t, history[1], Op{Index: 1, Process: int2pointer(1), Type: OpTypeOk, Value: txn1Mops}, "parse history, history[1]")
	assert.Equal(t, history[2], Op{Index: 2, Process: int2pointer(1), Type: OpTypeInvoke, Value: []Mop{Append{Key: "255", Value: 8}, Read{Key: "253", Value: nil}}}, "parse history, history[2]")
	assert.Equal(t, history[3], Op{Index: 3, Type: OpTypeOk, Value: []Mop{Append{Key: "255", Value: 8}, Read{Key: "253", Value: []int{1, 3, 4}}}}, "parse history, history[3]")
	assert.Equal(t, history[4], Op{Index: 4, Process: int2pointer(2), Type: OpTypeInvoke, Value: txn3MopsInvoke}, "parse history, history[4]")
	assert.Equal(t, history[5], Op{Index: 5, Type: OpTypeOk, Value: txn3MopsOk}, "parse history, history[5]")
	assert.Equal(t, history[6], Op{Index: 6, Type: OpTypeInvoke, Value: txn5MopsInvoke}, "parse history, history[6]")
	assert.Equal(t, history[7], Op{Index: 7, Type: OpTypeOk, Value: txn5MopsOk}, "parse history, history[7]")
}
