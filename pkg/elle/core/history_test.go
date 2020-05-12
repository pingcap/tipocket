package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const PaperExample = `{:index 0 :type :invoke  :value [[:append 253 1] [:append 253 3] [:append 253 4] [:append 255 2] [:append 255 3] [:append 255 4] [:append 255 5] [:append 256 1] [:append 256 2]]}
{:index 1 :type :ok      :process 1 :value [[:append 253 1] [:append 253 3] [:append 253 4] [:append 255 2] [:append 255 3] [:append 255 4] [:append 255 5] [:append 256 1] [:append 256 2]]}
{:index 2 :type :invoke, :process 1 :value [[:append 255 8] [:r 253 nil]]}
{:index 3 :type :ok,     :value [[:append 255 8] [:r 253 [1 3 4]]]}
{:index 4 :type :invoke, :process 2 :value [[:append 256 4] [:r 255 nil] [:r 256 nil] [:r 253 nil]]}
{:index 5 :type :ok,     :value [[:append 256 4] [:r 255 [2 3 4 5 8]] [:r 256 [1 2 4]] [:r 253 [1 3 4]]]}
{:index 6 :type :invoke, :value [[:append 250 10] [:r 253 nil] [:r 255 nil] [:append 256 3]]}
{:index 7 :type :ok      :value [[:append 250 10] [:r 253 [1 3 4]] [:r 255 [2 3 4 5]] [:append 256 3]]}`

func TestParseHistory(t *testing.T) {
	txn1Mops := []Mop{
		Append("253", 1),
		Append("253", 3),
		Append("253", 4),
		Append("255", 2),
		Append("255", 3),
		Append("255", 4),
		Append("255", 5),
		Append("256", 1),
		Append("256", 2),
	}
	txn3MopsInvoke := []Mop{
		Append("256", 4),
		Read("255", nil),
		Read("256", nil),
		Read("253", nil),
	}
	txn3MopsOk := []Mop{
		Append("256", 4),
		Read("255", []int{2, 3, 4, 5, 8}),
		Read("256", []int{1, 2, 4}),
		Read("253", []int{1, 3, 4}),
	}
	txn5MopsInvoke := []Mop{
		Append("250", 10),
		Read("253", nil),
		Read("255", nil),
		Append("256", 3),
	}
	txn5MopsOk := []Mop{
		Append("250", 10),
		Read("253", []int{1, 3, 4}),
		Read("255", []int{2, 3, 4, 5}),
		Append("256", 3),
	}

	history, err := ParseHistory(PaperExample)
	assert.Equal(t, err, nil, "parse history, no error")
	assert.Equal(t, len(history), 8, "parse history, length")
	assert.Equal(t, history[0], Op{Index: IntOptional{0}, Type: OpTypeInvoke, Value: &txn1Mops}, "parse history, history[0]")
	assert.Equal(t, history[1], Op{Index: IntOptional{1}, Process: NewOptInt(1), Type: OpTypeOk, Value: &txn1Mops}, "parse history, history[1]")
	assert.Equal(t, history[2], Op{Index: IntOptional{2}, Process: NewOptInt(1), Type: OpTypeInvoke, Value: &[]Mop{Append("255", 8), Read("253", nil)}}, "parse history, history[2]")
	assert.Equal(t, history[3], Op{Index: IntOptional{3}, Type: OpTypeOk, Value: &[]Mop{Append("255", 8), Read("253", []int{1, 3, 4})}}, "parse history, history[3]")
	assert.Equal(t, history[4], Op{Index: IntOptional{4}, Process: NewOptInt(2), Type: OpTypeInvoke, Value: &txn3MopsInvoke}, "parse history, history[4]")
	assert.Equal(t, history[5], Op{Index: IntOptional{5}, Type: OpTypeOk, Value: &txn3MopsOk}, "parse history, history[5]")
	assert.Equal(t, history[6], Op{Index: IntOptional{6}, Type: OpTypeInvoke, Value: &txn5MopsInvoke}, "parse history, history[6]")
	assert.Equal(t, history[7], Op{Index: IntOptional{7}, Type: OpTypeOk, Value: &txn5MopsOk}, "parse history, history[7]")
}
