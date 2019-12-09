package sqlsmith

import (
	"testing"

	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/opcode"
)

func TestSQLSmith_Walker(t *testing.T) {
	ss := New()
	ss.LoadSchema([][5]string{
		// tasks table
		{"community", "tasks", "BASE TABLE", "id", "int(11)"},
		{"community", "tasks", "BASE TABLE", "season", "int(11)"},
		{"community", "tasks", "BASE TABLE", "complete_user", "varchar(255)"},
		{"community", "tasks", "BASE TABLE", "complete_team", "int(11)"},
		{"community", "tasks", "BASE TABLE", "owner", "varchar(255)"},
		{"community", "tasks", "BASE TABLE", "repo", "varchar(255)"},
		{"community", "tasks", "BASE TABLE", "title", "varchar(1023)"},
		{"community", "tasks", "BASE TABLE", "issue_number", "int(11)"},
		{"community", "tasks", "BASE TABLE", "pull_number", "int(11)"},
		{"community", "tasks", "BASE TABLE", "level", "varchar(255)"},
		{"community", "tasks", "BASE TABLE", "min_score", "int(11)"},
		{"community", "tasks", "BASE TABLE", "score", "int(11)"},
		{"community", "tasks", "BASE TABLE", "status", "varchar(255)"},
		{"community", "tasks", "BASE TABLE", "created_at", "timestamp"},
		{"community", "tasks", "BASE TABLE", "expired", "varchar(255)"},
		// picks table
		{"community", "picks", "BASE TABLE", "id", "int(11)"},
		{"community", "picks", "BASE TABLE", "season", "int(11)"},
		{"community", "picks", "BASE TABLE", "task_id", "int(11)"},
		{"community", "picks", "BASE TABLE", "teamID", "int(11)"},
		{"community", "picks", "BASE TABLE", "user", "varchar(255)"},
		{"community", "picks", "BASE TABLE", "pull_number", "int(11)"},
		{"community", "picks", "BASE TABLE", "status", "varchar(255)"},
		{"community", "picks", "BASE TABLE", "created_at", "timestamp"},
		{"community", "picks", "BASE TABLE", "updated_at", "timestamp"},
		{"community", "picks", "BASE TABLE", "closed_at", "timestamp"},
	})

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
