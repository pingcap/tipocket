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


// CreateTableStmt create table
func (s *SQLSmith) CreateTableStmt() (string, error) {
	tree := s.createTableStmt()
	return s.Walk(tree)
}

// AlterTableStmt alter table
func (s *SQLSmith) AlterTableStmt() (string, error) {
	tree := s.alterTableStmt()
	return s.Walk(tree)
}

// CreateIndexStmt create index
func (s *SQLSmith) CreateIndexStmt() (string, error) {
	tree := s.createIndexStmt()
	return s.Walk(tree)
}
