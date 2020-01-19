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
	"testing"
)


func TestSQLSmith_Select(t *testing.T) {
	ss := New()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)
	for i := 0; i < 500; i++ {
		sql, err := ss.SelectStmt(1 + rand.Intn(5))
		if err != nil {
			t.Log(sql, err)
		}
	}
	sql, _ := ss.SelectStmt(6)
	t.Log(sql)
}

func TestSQLSmith_Update(t *testing.T) {
	ss := New()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)

	for i := 0; i < 1000; i++ {
		sql, err := ss.UpdateStmt()
		if err != nil {
			t.Log(sql, err)
		}
	}
	sql, _ := ss.UpdateStmt()
	t.Log(sql)
}

func TestSQLSmith_Insert(t *testing.T) {
	ss := New()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)

	for i := 0; i < 1000; i++ {
		sql, err := ss.InsertStmtAST()
		if err != nil {
			t.Log(sql, err)
		}
	}
	sql, err := ss.InsertStmtAST()
	t.Log(sql, err)
}
