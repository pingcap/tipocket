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

package sqlsmith

import (
	"math/rand"
	"time"

	"github.com/juju/errors"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/tipocket/go-sqlsmith/stateflow"
	"github.com/pingcap/tipocket/go-sqlsmith/types"
	"github.com/pingcap/tipocket/go-sqlsmith/util"
	"github.com/pingcap/tipocket/pocket/pkg/generator/generator"

	// _ "github.com/pingcap/tidb/types/parser_driver"
)

// SQLSmith defines SQLSmith struct
type SQLSmith struct {
	depth int
	maxDepth int
	Rand *rand.Rand
	Databases map[string]*types.Database
	subTableIndex int
	Node ast.Node
	currDB string
	debug bool
	stable bool
}

// New create SQLSmith instance
func New() generator.Generator {
	return new()
}

func new() *SQLSmith {
	return &SQLSmith{
		Rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
		Databases: make(map[string]*types.Database),
	}
}

// Debug turn on debug mode
func (s *SQLSmith) Debug() {
	s.debug = true
}

// SetDB set current database
func (s *SQLSmith) SetDB(db string) {
	s.currDB = db
}

// GetCurrDBName returns current selected dbname
func (s *SQLSmith) GetCurrDBName() string {
	return s.currDB
}

// GetDB get current database without nil
func (s *SQLSmith) GetDB(db string) *types.Database {
	if db, ok := s.Databases[db]; ok {
		return db
	}
	return &types.Database{
		Name: db,
	}
}

// Stable set generated SQLs no rand
func (s *SQLSmith) Stable() {
	s.stable = true
}

// SetStable set stable to given value
func (s *SQLSmith) SetStable(stable bool) {
	s.stable = stable
}

// Walk will walk the tree and fillin tables and columns data
func (s *SQLSmith) Walk(tree ast.Node) (string, string, error) {
	node, table, err := stateflow.New(s.GetDB(s.currDB), s.stable).WalkTree(tree)
	if err != nil {
		return "", "", errors.Trace(err)
	}
	s.debugPrintf("node AST %+v\n", node)
	sql, err := util.BufferOut(node)
	return sql, table, errors.Trace(err)
}
