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
	"testing"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/generator/generator"
)

// TestSQLSmith_CreateTable tests create table statement
func TestSQLSmith_CreateTable(t *testing.T) {
	ss := New()
	ss.LoadSchema([][5]string{}, make(map[string][]string))
	ss.SetDB(dbname)
	sql, _, _ := ss.CreateTableStmt()
	t.Log(sql)
}

// TestSQLSmith_AlterTable tests alter table statement
func TestSQLSmith_AlterTable(t *testing.T) {
	ss := New()
	indexes["users"] = []string{"idx1", "idx2"}
	ss.LoadSchema(schema, indexes)
	ss.SetDB(dbname)

	sql, _ := ss.AlterTableStmt(&generator.DDLOptions{OnlineDDL: true})
	t.Log(sql)
}

// TestSQLSmith_CreateIndex tests create index statement
func TestSQLSmith_CreateIndex(t *testing.T) {
	ss := New()
	ss.LoadSchema(schema, indexes)
	ss.SetDB(dbname)

	sql, _ := ss.CreateIndexStmt(&generator.DDLOptions{OnlineDDL: true})
	t.Log(sql)
}
