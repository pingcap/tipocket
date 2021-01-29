// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

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
	assert.False(t, o.Has(100))
	assert.True(t, o.Has(11))
	orders = []int{}
	for o.Next() {
		orders = append(orders, o.Val())
	}
	assert.Equal(t, []int{10, 11}, orders)
}
