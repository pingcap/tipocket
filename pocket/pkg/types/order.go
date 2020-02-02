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
	"sync"
)

// Without general type, the "container/list" will not be graceful

// Order for log executor orders
type Order struct {
	sync.Mutex
	history []int
	index   int
}

// NewOrder create Order
func NewOrder() *Order {
	o := &Order{
		history: []int{},
		index:   -1,
	}
	return o
}

// Push push a history, will move the id to end of
func (o *Order) Push(id int) {
	o.Lock()
	defer o.Unlock()
	for index, i := range o.history {
		if i == id {
			if index == len(o.history)-1 {
				return
			}
			o.history = append(o.history[:index], o.history[index+1:]...)
		}
	}
	o.history = append(o.history, id)
}

// Start reset index
func (o *Order) Start() {
	o.index = -1
}

// Next is for walking through
func (o *Order) Next() bool {
	if o.index == -1 {
		o.Lock()
	}
	o.index++
	if o.index == len(o.history) {
		o.Start()
		o.Reset()
		o.Unlock()
		return false
	}
	return true
}

// Val return value for current index
func (o *Order) Val() int {
	return o.history[o.index]
}

// Reset history
func (o *Order) Reset() {
	o.history = []int{}
}

// GetHistroy get copied history slice
func (o *Order) GetHistroy() []int {
	h := o.history
	return h
}

// Has given value in order list
func (o *Order) Has(i int) bool {
	o.Lock()
	defer o.Unlock()
	for _, h := range o.history {
		if i == h {
			return true
		}
	}
	return false
}
