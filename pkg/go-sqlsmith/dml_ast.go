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
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/opcode"

	"github.com/pingcap/tipocket/pkg/go-sqlsmith/util"
)

func (s *SQLSmith) selectStmt(depth int) ast.Node {
	selectStmtNode := ast.SelectStmt{
		SelectStmtOpts: &ast.SelectStmtOpts{
			SQLCache: true,
		},
		Fields: &ast.FieldList{
			Fields: []*ast.SelectField{},
		},
		OrderBy: &ast.OrderByClause{},
	}

	if depth <= 1 {
		complex := 0
		if util.Rd(3) == 0 {
			complex = 1
		}
		selectStmtNode.Where = s.binaryOperationExpr(0, complex)
	} else {
		selectStmtNode.Where = s.binaryOperationExpr(util.Rd(depth), 1)
	}

	selectStmtNode.TableHints = s.tableHintsExpr()
	selectStmtNode.From = s.tableRefsClause(depth)

	return &selectStmtNode
}

func (s *SQLSmith) selectForUpdateStmt(depth int) ast.Node {
	node := s.selectStmt(depth)
	node.(*ast.SelectStmt).LockTp = ast.SelectLockForUpdate
	return node
}

func (s *SQLSmith) updateStmt() ast.Node {
	updateStmtNode := ast.UpdateStmt{
		List: []*ast.Assignment{},
		TableRefs: &ast.TableRefsClause{
			TableRefs: &ast.Join{
				Left: &ast.TableName{},
			},
		},
	}

	whereRand := s.rd(10)
	if whereRand < 8 {
		updateStmtNode.Where = s.binaryOperationExpr(whereRand, 0)
	} else {
		updateStmtNode.Where = ast.NewValueExpr(1, "", "")
	}

	updateStmtNode.TableHints = s.tableHintsExpr()

	return &updateStmtNode
}

func (s *SQLSmith) insertStmt() ast.Node {
	insertStmtNode := ast.InsertStmt{
		Table: &ast.TableRefsClause{
			TableRefs: &ast.Join{
				Left: &ast.TableName{},
			},
		},
		Lists:   [][]ast.ExprNode{},
		Columns: []*ast.ColumnName{},
	}
	return &insertStmtNode
}

func (s *SQLSmith) deleteStmt() ast.Node {
	deleteStmtNode := ast.DeleteStmt{
		TableRefs: s.tableRefsClause(1),
	}

	whereRand := s.rd(10)
	if whereRand < 8 {
		deleteStmtNode.Where = s.binaryOperationExpr(whereRand, 0)
	} else {
		deleteStmtNode.Where = ast.NewValueExpr(1, "", "")
	}

	deleteStmtNode.TableHints = s.tableHintsExpr()

	return &deleteStmtNode
}

func (s *SQLSmith) tableHintsExpr() (hints []*ast.TableOptimizerHint) {
	if !s.Hint() {
		return
	}
	length := 0
	switch n := util.Rd(7)*util.Rd(7) - 17; {
	case n <= 0:
		length = 0
	case n < 4:
		length = 1
	case n < 9:
		length = 2
	case n < 14:
		length = 3
	default:
		length = 4
	}
	for i := 0; i < length; i++ {
		hints = append(hints, &ast.TableOptimizerHint{})
	}
	return
}

func (s *SQLSmith) tableRefsClause(depth int) *ast.TableRefsClause {
	tableRefsClause := ast.TableRefsClause{
		TableRefs: &ast.Join{
			Left: &ast.TableName{},
		},
	}

	if depth > 1 {
		// if s.rd(100) > 50 {
		// 	tableRefsClause.TableRefs.Right = &ast.TableName{}
		// } else {
		// 	tableRefsClause.TableRefs.Right = &ast.TableSource{
		// 		Source: s.selectStmt(depth + 1),
		// 	}
		// }
		tableRefsClause.TableRefs.Right = &ast.TableSource{
			Source: s.selectStmt(depth - 1),
		}
		if s.rd(100) > 30 {
			tableRefsClause.TableRefs.On = &ast.OnCondition{
				Expr: &ast.BinaryOperationExpr{
					Op: opcode.EQ,
					L:  &ast.ColumnNameExpr{},
					R:  &ast.ColumnNameExpr{},
				},
			}
		}
	}

	return &tableRefsClause
}

func (s *SQLSmith) binaryOperationExpr(depth, complex int) ast.ExprNode {
	node := ast.BinaryOperationExpr{}
	if depth > 0 {
		r := util.Rd(4)
		switch r {
		case 0:
			node.Op = opcode.LogicXor
		case 1:
			node.Op = opcode.LogicOr
		default:
			node.Op = opcode.LogicAnd
		}
		node.L = s.binaryOperationExpr(depth-1, complex)
		node.R = s.binaryOperationExpr(0, complex)
	} else {
		if complex > 0 {
			switch util.Rd(4) {
			case 0:
				return s.patternInExpr()
			default:
				switch util.Rd(4) {
				case 0:
					node.Op = opcode.GT
				case 1:
					node.Op = opcode.LT
				case 2:
					node.Op = opcode.NE
				default:
					node.Op = opcode.EQ
				}
				node.L = s.exprNode(false)
				node.R = s.exprNode(true)
			}
		} else {
			switch util.Rd(4) {
			case 0:
				node.Op = opcode.GT
			case 1:
				node.Op = opcode.LT
			case 2:
				node.Op = opcode.NE
			default:
				node.Op = opcode.EQ
			}
			node.L = &ast.ColumnNameExpr{}
			node.R = ast.NewValueExpr(1, "", "")
		}
	}
	return &node
}

func (s *SQLSmith) patternInExpr() *ast.PatternInExpr {
	// expr := s.exprNode()
	// switch node := expr.(type) {
	// case *ast.SubqueryExpr:
	// 	// may need refine after fully support of ResultSetNode interface
	// 	node.Query.(*ast.SelectStmt).Limit = &ast.Limit {
	// 		Count: ast.NewValueExpr(1, "", ""),
	// 	}
	// }

	return &ast.PatternInExpr{
		Expr: &ast.ColumnNameExpr{},
		Sel:  s.subqueryExpr(),
	}
}

func (s *SQLSmith) subqueryExpr() *ast.SubqueryExpr {
	return &ast.SubqueryExpr{
		Query:     s.selectStmt(1),
		MultiRows: true,
	}
}

func (s *SQLSmith) exprNode(cons bool) ast.ExprNode {
	switch util.Rd(6) {
	case 0:
		return &ast.ColumnNameExpr{}
	case 1:
		return s.subqueryExpr()
	default:
		// hope there is an empty value type
		if cons {
			return ast.NewValueExpr(1, "", "")
		}
		return &ast.ColumnNameExpr{}
	}
	// panic("unhandled switch")
}

// func (s *SQLSmith) whereExprNode(depth int) ast.ExprNode {
// 	whereCount := util.Rd(4)
// 	if whereCount == 0 {
// 		return nil
// 	}
// 	var binaryOperation *ast.BinaryOperationExpr
// 	for i := 0; i < whereCount; i++ {
// 		binaryOperation =
// 	}
// 	return binaryOperation
// }

// func (s *SQLSmith) whereExprNode(depth int)
