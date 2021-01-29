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

package builtin

import "github.com/pingcap/parser/ast"

var informationFunctions = []*functionClass{
	// will make diff
	// {ast.ConnectionID, 0, 0, false, true, false},
	{ast.CurrentUser, 0, 0, false, true, true},
	// should be fix
	// {ast.CurrentRole, 0, 0, false, true, false},
	{ast.Database, 0, 0, false, true, false},
	// This function is a synonym for DATABASE().
	// See http://dev.mysql.com/doc/refman/5.7/en/information-functions.html#function_schema
	{ast.Schema, 0, 0, false, true, false},
	{ast.FoundRows, 0, 0, false, true, false},
	{ast.LastInsertId, 0, 1, false, true, false},
	{ast.User, 0, 0, false, true, true},
	{ast.Version, 0, 0, false, true, true},
	{ast.Benchmark, 2, 2, false, true, false},
	// {ast.Charset, 1, 1, false, true, false},
	// {ast.Coercibility, 1, 1, false, true, false},
	// {ast.Collation, 1, 1, false, true, false},
	{ast.RowCount, 0, 0, false, true, false},
	// Will make difference in abtest
	// {ast.SessionUser, 0, 0, false, true, false},
	// {ast.SystemUser, 0, 0, false, true, false},
}
