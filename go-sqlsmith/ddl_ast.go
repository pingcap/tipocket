package sqlsmith

import (
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/parser/model"
	"github.com/pingcap/tipocket/go-sqlsmith/util"
)

func (s *SQLSmith) createTableStmt() ast.Node {
	createTableNode := ast.CreateTableStmt{
		Table: &ast.TableName{},
		Cols: []*ast.ColumnDef{},
		Constraints: []*ast.Constraint{},
		Options: []*ast.TableOption{},
	}
	if util.Rd(4) == 0 {
		createTableNode.Partition = s.partitionStmt()
	}

	return &createTableNode
}

func (s *SQLSmith) alterTableStmt() ast.Node {
	return &ast.AlterTableStmt{
		Table: &ast.TableName{},
		Specs: []*ast.AlterTableSpec{
			s.alterTableSpec(),
		},
	}
}

func (s *SQLSmith) partitionStmt() *ast.PartitionOptions{
	return &ast.PartitionOptions{
		PartitionMethod: ast.PartitionMethod{
			ColumnNames: []*ast.ColumnName{},
		},
		Definitions: []*ast.PartitionDefinition{},
	}
}

func (s *SQLSmith) alterTableSpec() *ast.AlterTableSpec {
	switch util.Rd(4) {
	case 0:
		return s.alterTableSpecDropColumn()
	case 1:
		return s.alterTableSpecDropIndex()
	default:
		return s.alterTableSpecAddColumns()
	}
}

func (s *SQLSmith) alterTableSpecAddColumns() *ast.AlterTableSpec {
	return &ast.AlterTableSpec{
		Tp: ast.AlterTableAddColumns,
		NewColumns: []*ast.ColumnDef{{}},
	}
}

func (s *SQLSmith) alterTableSpecDropIndex() *ast.AlterTableSpec {
	return &ast.AlterTableSpec{
		Tp: ast.AlterTableDropIndex,
	}
}

func (s *SQLSmith) alterTableSpecDropColumn() *ast.AlterTableSpec {
	return &ast.AlterTableSpec{
		Tp: ast.AlterTableDropColumn,
		OldColumnName: &ast.ColumnName{},
	}
}

func (s *SQLSmith) createIndexStmt() *ast.CreateIndexStmt {
	var indexType model.IndexType
	switch util.Rd(2) {
	case 0:
		indexType = model.IndexTypeBtree
	default:
		indexType = model.IndexTypeHash
	}

	node := ast.CreateIndexStmt{
		Table: &ast.TableName{},
		IndexColNames: []*ast.IndexColName{},
		IndexOption: &ast.IndexOption{
			Tp: indexType,
		},
	}
	return &node
}
