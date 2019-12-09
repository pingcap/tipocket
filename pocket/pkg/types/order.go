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
		index: -1,
	}
	return o
}

// Push push a history, will move the id to end of 
func (o *Order)Push(id int) {
	o.Lock()
	defer o.Unlock()
	for index, i := range o.history {
		if i == id {
			if index == len(o.history) - 1 {
				return
			}
			o.history = append(o.history[:index], o.history[index+1:]...)
		}
	}
	o.history = append(o.history, id)
}

// Start reset index
func (o *Order)Start() {
	o.index = -1
}

// Next is for walking through
func (o *Order)Next() bool {
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
func (o *Order)Val() int {
	return o.history[o.index]
}

// Reset history
func (o *Order) Reset () {
	o.history = []int{}
}
