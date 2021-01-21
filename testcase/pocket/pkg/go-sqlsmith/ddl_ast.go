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
	"github.com/pingcap/parser/model"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/go-sqlsmith/util"
)

func (s *SQLSmith) createTableStmt() ast.Node {
	createTableNode := ast.CreateTableStmt{
		Table:       &ast.TableName{},
		Cols:        []*ast.ColumnDef{},
		Constraints: []*ast.Constraint{},
		Options:     []*ast.TableOption{},
	}
	// TODO: config for enable partition
	// partitionStmt is disabled
	// createTableNode.Partition = s.partitionStmt()

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

func (s *SQLSmith) partitionStmt() *ast.PartitionOptions {
	return &ast.PartitionOptions{
		PartitionMethod: ast.PartitionMethod{
			ColumnNames: []*ast.ColumnName{},
		},
		Definitions: []*ast.PartitionDefinition{},
	}
}

func (s *SQLSmith) alterTableSpec() *ast.AlterTableSpec {
	switch util.Rd(2) {
	case 0:
		return s.alterTableSpecAddColumns()
	default:
		return s.alterTableSpecDropIndex()
	}
	// switch util.Rd(4) {
	// case 0:
	// 	return s.alterTableSpecDropColumn()
	// case 1:
	// 	return s.alterTableSpecDropIndex()
	// default:
	// 	// return s.alterTableSpecAddColumns()
	// 	return s.alterTableSpecDropColumn()
	// }
}

func (s *SQLSmith) alterTableSpecAddColumns() *ast.AlterTableSpec {
	return &ast.AlterTableSpec{
		Tp:         ast.AlterTableAddColumns,
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
		Tp:            ast.AlterTableDropColumn,
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
		Table:                   &ast.TableName{},
		IndexPartSpecifications: []*ast.IndexPartSpecification{},
		IndexOption: &ast.IndexOption{
			Tp: indexType,
		},
	}
	return &node
}
