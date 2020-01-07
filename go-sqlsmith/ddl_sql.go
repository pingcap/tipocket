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
	"github.com/pingcap/tipocket/pocket/pkg/generator/generator"
)

// CreateTableStmt create table
func (s *SQLSmith) CreateTableStmt() (string, error) {
	tree := s.createTableStmt()
	stmt, _, err := s.Walk(tree)
	return stmt, err
}

// AlterTableStmt alter table
func (s *SQLSmith) AlterTableStmt(opt *generator.DDLOptions) (string, error) {
	s.setOnlineOtherTables(opt)
	defer s.freeOnlineOtherTables()
	tree := s.alterTableStmt()
	stmt, _, err := s.Walk(tree)
	return stmt, err
}

// CreateIndexStmt create index
func (s *SQLSmith) CreateIndexStmt(opt *generator.DDLOptions) (string, error) {
	s.setOnlineOtherTables(opt)
	defer s.freeOnlineOtherTables()
	tree := s.createIndexStmt()
	stmt, _, err := s.Walk(tree)
	return stmt, err
}
