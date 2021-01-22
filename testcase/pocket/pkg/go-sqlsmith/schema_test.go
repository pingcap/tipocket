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

	"github.com/stretchr/testify/assert"
)

// test schema
var (
	schema = [][5]string{
		{"community", "comments", "BASE TABLE", "id", "int(11)"},
		{"community", "comments", "BASE TABLE", "owner", "varchar(255)"},
		{"community", "comments", "BASE TABLE", "repo", "varchar(255)"},
		{"community", "comments", "BASE TABLE", "comment_id", "int(11)"},
		{"community", "comments", "BASE TABLE", "comment_type", "varchar(128)"},
		{"community", "comments", "BASE TABLE", "pull_number", "int(11)"},
		{"community", "comments", "BASE TABLE", "body", "text"},
		{"community", "comments", "BASE TABLE", "user", "varchar(255)"},
		{"community", "comments", "BASE TABLE", "url", "varchar(1023)"},
		{"community", "comments", "BASE TABLE", "association", "varchar(255)"},
		{"community", "comments", "BASE TABLE", "relation", "varchar(255)"},
		{"community", "comments", "BASE TABLE", "created_at", "timestamp"},
		{"community", "comments", "BASE TABLE", "updated_at", "timestamp"},
		{"community", "picks", "BASE TABLE", "id", "int(11)"},
		{"community", "picks", "BASE TABLE", "season", "int(11)"},
		{"community", "picks", "BASE TABLE", "task_id", "int(11)"},
		{"community", "picks", "BASE TABLE", "teamID", "int(11)"},
		{"community", "picks", "BASE TABLE", "user", "varchar(255)"},
		{"community", "picks", "BASE TABLE", "pull_number", "int(11)"},
		{"community", "picks", "BASE TABLE", "status", "varchar(128)"},
		{"community", "picks", "BASE TABLE", "created_at", "timestamp"},
		{"community", "picks", "BASE TABLE", "updated_at", "timestamp"},
		{"community", "picks", "BASE TABLE", "closed_at", "datetime"},
		{"community", "pulls", "BASE TABLE", "id", "int(11)"},
		{"community", "pulls", "BASE TABLE", "owner", "varchar(255)"},
		{"community", "pulls", "BASE TABLE", "repo", "varchar(255)"},
		{"community", "pulls", "BASE TABLE", "pull_number", "int(11)"},
		{"community", "pulls", "BASE TABLE", "title", "text"},
		{"community", "pulls", "BASE TABLE", "body", "text"},
		{"community", "pulls", "BASE TABLE", "user", "varchar(255)"},
		{"community", "pulls", "BASE TABLE", "association", "varchar(255)"},
		{"community", "pulls", "BASE TABLE", "relation", "varchar(255)"},
		{"community", "pulls", "BASE TABLE", "label", "text"},
		{"community", "pulls", "BASE TABLE", "status", "varchar(128)"},
		{"community", "pulls", "BASE TABLE", "created_at", "timestamp"},
		{"community", "pulls", "BASE TABLE", "updated_at", "timestamp"},
		{"community", "pulls", "BASE TABLE", "closed_at", "datetime"},
		{"community", "pulls", "BASE TABLE", "merged_at", "datetime"},
		{"community", "tasks", "BASE TABLE", "id", "int(11)"},
		{"community", "tasks", "BASE TABLE", "season", "int(11)"},
		{"community", "tasks", "BASE TABLE", "complete_user", "varchar(255)"},
		{"community", "tasks", "BASE TABLE", "complete_team", "int(11)"},
		{"community", "tasks", "BASE TABLE", "owner", "varchar(255)"},
		{"community", "tasks", "BASE TABLE", "repo", "varchar(255)"},
		{"community", "tasks", "BASE TABLE", "title", "varchar(2047)"},
		{"community", "tasks", "BASE TABLE", "issue_number", "int(11)"},
		{"community", "tasks", "BASE TABLE", "pull_number", "int(11)"},
		{"community", "tasks", "BASE TABLE", "level", "varchar(255)"},
		{"community", "tasks", "BASE TABLE", "min_score", "int(11)"},
		{"community", "tasks", "BASE TABLE", "score", "int(11)"},
		{"community", "tasks", "BASE TABLE", "status", "varchar(255)"},
		{"community", "tasks", "BASE TABLE", "created_at", "timestamp"},
		{"community", "tasks", "BASE TABLE", "expired", "varchar(255)"},
		{"community", "teams", "BASE TABLE", "id", "int(11)"},
		{"community", "teams", "BASE TABLE", "season", "int(11)"},
		{"community", "teams", "BASE TABLE", "name", "varchar(255)"},
		{"community", "teams", "BASE TABLE", "issue_url", "varchar(1023)"},
		{"community", "users", "BASE TABLE", "id", "int(11)"},
		{"community", "users", "BASE TABLE", "season", "int(11)"},
		{"community", "users", "BASE TABLE", "user", "varchar(255)"},
		{"community", "users", "BASE TABLE", "team_id", "int(11)"},
	}
	dbname  = "community"
	indexes = make(map[string][]string)
	tables  = []string{"comments", "picks", "pulls", "tasks", "teams", "users"}
)

// TestSQLSmith_LoadIndexes tests with load table indexes
func TestSQLSmith_LoadIndexes(t *testing.T) {
	ss := new()
	indexes["users"] = []string{"idx1", "idx2"}
	ss.LoadSchema(schema, indexes)
	ss.SetDB(dbname)

	assert.Equal(t, len(ss.Databases[dbname].Tables), 6)
	assert.Equal(t, len(ss.Databases[dbname].Tables["users"].Indexes), 2)
}
