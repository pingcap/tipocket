package sqlsmith

import (
	"testing"

	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/opcode"
)

func TestSQLSmith_Walker(t *testing.T) {
	ss := New()
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
				Left: &ast.TableName{},
				Right: &joinSub,
				On: &ast.OnCondition{
					Expr: &ast.BinaryOperationExpr{
						Op: opcode.EQ,
						L: &ast.ColumnNameExpr{},
						R: &ast.ColumnNameExpr{},
					},
				},
			},
		},
	}

	ss.SetDB("community")
	sql, err :=	ss.Walk(&node)

	if err != nil {
		t.Fatalf("walk error %v", err)
	}
	t.Log(sql)
}
