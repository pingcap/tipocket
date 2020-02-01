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

// TestSQLSmith_Select tests select statment
func TestSQLSmith_Select(t *testing.T) {
	ss := new()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)
	for i := 0; i < 500; i++ {
		sql, _, err := ss.SelectStmt(1 + rand.Intn(5))
		if err != nil {
			t.Log(sql, err)
		}
	}
	sql, _, _ := ss.SelectStmt(1)
	t.Log(sql)
}

// TestSQLSmith_SelectForUpdateStmt tests select for update statement
func TestSQLSmith_SelectForUpdateStmt(t *testing.T) {
	ss := new()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)
	for i := 0; i < 500; i++ {
		sql, _, err := ss.SelectForUpdateStmt(1 + rand.Intn(5))
		if err != nil {
			t.Log(sql, err)
		}
	}
	sql, _, _ := ss.SelectForUpdateStmt(6)
	t.Log(sql)
}

// TestSQLSmith_Update tests update statement
func TestSQLSmith_Update(t *testing.T) {
	ss := new()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)

	for i := 0; i < 1000; i++ {
		sql, _, err := ss.UpdateStmt()
		if err != nil {
			t.Log(sql, err)
		}
	}
	sql, _, _ := ss.UpdateStmt()
	t.Log(sql)
}

// TestSQLSmith_Insert tests insert statment
func TestSQLSmith_Insert(t *testing.T) {
	ss := new()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)

	for i := 0; i < 1000; i++ {
		sql, _, err := ss.InsertStmt(false)
		if err != nil {
			t.Log(sql, err)
		}
	}
	sql, _, err := ss.InsertStmt(false)
	t.Log(sql, err)
}

// TestSQLSmith_Delete tests delete statment
func TestSQLSmith_Delete(t *testing.T) {
	ss := new()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)

	for i := 0; i < 1000; i++ {
		sql, _, err := ss.DeleteStmt()
		if err != nil {
			t.Log(sql, err)
		}
	}
	sql, _, err := ss.DeleteStmt()
	t.Log(sql, err)
}
