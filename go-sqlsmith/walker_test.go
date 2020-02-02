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

	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/opcode"
)

func TestSQLSmith_Walker(t *testing.T) {
	ss := new()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)

	// walker should fill in
	// ast.FieldList
	// ast.ColumnNameExpr
	// ast.TableName

	joinSub := ast.TableSource{
		Source: &ast.SelectStmt{
			SelectStmtOpts: &ast.SelectStmtOpts{
				SQLCache: true,
			},
			Fields: &ast.FieldList{
				Fields: []*ast.SelectField{},
			},
			From: &ast.TableRefsClause{
				TableRefs: &ast.Join{
					Left: &ast.TableName{},
				},
			},
		},
	}

	node := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{
			SQLCache: true,
		},
		Fields: &ast.FieldList{
			Fields: []*ast.SelectField{},
		},
		From: &ast.TableRefsClause{
			TableRefs: &ast.Join{
				Left:  &ast.TableName{},
				Right: &joinSub,
				On: &ast.OnCondition{
					Expr: &ast.BinaryOperationExpr{
						Op: opcode.EQ,
						L:  &ast.ColumnNameExpr{},
						R:  &ast.ColumnNameExpr{},
					},
				},
			},
		},
	}

	ss.SetDB("community")
	sql, _, err := ss.Walk(&node)

	if err != nil {
		t.Fatalf("walk error %v", err)
	}
	t.Log(sql)
}
