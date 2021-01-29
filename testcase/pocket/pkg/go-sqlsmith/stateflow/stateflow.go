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
	"math/rand"
	"time"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/go-sqlsmith/types"
)

// StateFlow defines the struct
type StateFlow struct {
	// db is a ref took from SQLSmith, should never change it
	db         *types.Database
	rand       *rand.Rand
	tableIndex int
	stable     bool
}

// New Create StateFlow
func New(db *types.Database, stable bool) *StateFlow {
	// copy whole db here may cost time, but ensure the global safety
	// maybe a future TODO
	return &StateFlow{
		db:     db,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		stable: stable,
	}
}
