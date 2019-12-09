package types

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestOrder(t *testing.T) {
	o := NewOrder()
	o.Push(11)
	o.Push(12)
	o.Push(13)
	o.Push(14)
	o.Push(12)
	o.Push(12)

	orders := []int{}
	for o.Next() {
		orders = append(orders, o.Val())
	}
	assert.Equal(t, []int{11, 13, 14, 12}, orders)

	o.Push(11)
	o.Push(10)
	o.Push(11)
	o.Push(11)
	orders = []int{}
	for o.Next() {
		orders = append(orders, o.Val())
	}
	assert.Equal(t, []int{10, 11}, orders)
}
