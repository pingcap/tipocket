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
	"time"
)

// Log line struct
type Log struct {
	Time  time.Time
	Node  int
	SQL   *SQL
	State string
}

// ByLog implement sort interface for log
type ByLog []*Log

func (a ByLog) Len() int      { return len(a) }
func (a ByLog) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLog) Less(i, j int) bool {
	return a[i].GetTime().Before(a[j].GetTime())
}

// GetTime get log time
func (l *Log) GetTime() time.Time {
	return l.Time
}

// GetNode get executor id
func (l *Log) GetNode() int {
	return l.Node
}

// GetSQL get SQL struct
func (l *Log) GetSQL() *SQL {
	if l.SQL != nil {
		return l.SQL
	}
	return &SQL{}
}
