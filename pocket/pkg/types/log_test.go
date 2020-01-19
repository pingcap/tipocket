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
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const timeLayout = `2006/01/02 15:04:05.000 -07:00`

func TestSQLLogSort(t *testing.T) {
	var logs = []*Log{
		{
			Time: mustParse("2006-01-02 15:04:05", "2019-08-10 11:45:16"),
			SQL: &SQL{
				SQLType: SQLTypeDMLSelect,
				SQLStmt: "SELECT * FROM t",
			},
		},
		{
			Time: mustParse("2006-01-02 15:04:05", "2019-08-10 11:45:15"),
			SQL: &SQL{
				SQLType: SQLTypeDMLUpdate,
				SQLStmt: "UPDATE t SET c = 1",
			},
		},
		{
			Time: mustParse("2006-01-02 15:04:05", "2019-08-10 11:45:14"),
			SQL: &SQL{
				SQLType: SQLTypeDMLInsert,
				SQLStmt: "INSERT INTO t(c) VALUES(1), (2)",
			},
		},
		{
			Time: mustParse("2006-01-02 15:04:05", "2019-08-10 11:45:17"),
			SQL: &SQL{
				SQLType: SQLTypeDMLDelete,
				SQLStmt: "DELETE FROM t",
			},
		},
	}
	sort.Sort(ByLog(logs))

	assert.Equal(t, logs[0].GetSQL().SQLType, SQLTypeDMLInsert)
	assert.Equal(t, logs[1].GetSQL().SQLType, SQLTypeDMLUpdate)
	assert.Equal(t, logs[2].GetSQL().SQLType, SQLTypeDMLSelect)
	assert.Equal(t, logs[3].GetSQL().SQLType, SQLTypeDMLDelete)
}

func mustParse(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	if err != nil {
		panic("parse failed")
	}
	return t
}
