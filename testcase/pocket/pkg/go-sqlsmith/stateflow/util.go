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
