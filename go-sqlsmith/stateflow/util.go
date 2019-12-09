package stateflow

import (
	"fmt"
	"runtime/debug"
)

func (s *StateFlow) getSubTableName() string {
	name := fmt.Sprintf("s%d", s.tableIndex)
	s.tableIndex++
	return name
}

func (s *StateFlow) shouldNotWalkHere(a ...interface{}) {
	fmt.Println("should not walk here")
	fmt.Println(a...)
	debug.PrintStack()
}