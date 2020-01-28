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

package stateflow


import (
	"fmt"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/tipocket/go-sqlsmith/types"
	"github.com/pingcap/tipocket/go-sqlsmith/util"
)

func (s *StateFlow) randTableFromTable(table *types.Table, newName bool, fn bool) (*types.Table) {
	newTableName := ""
	if newName {
		newTableName = s.getSubTableName()
	}
	columns := s.randColumns(table)
	newTable := types.Table{
		DB: table.DB,
		Type: table.Type,
		Columns: make(map[string]*types.Column),
	}
	if newName {
		newTable.OriginTable = table.Table
		newTable.Table = newTableName
	} else {
		newTable.Table = table.Table
	}
	for _, column := range columns {
		newColumn := &types.Column{
			DB: column.DB,
			Table: column.Table,
			OriginTable: column.OriginTable,
			Column: column.Column,
			OriginColumn: column.OriginColumn,
			DataType: column.DataType,
			Func: column.Func,
		}
		if newName {
			newColumn.Table = newTableName
		} else {
			newColumn.Table = table.Table
		}
		newTable.Columns[newColumn.Column] = newColumn
	}
	if fn {
		r := util.Rd(4)
		for i := 0; i < r; i++ {
			k := fmt.Sprintf("f%d", i)
			newTable.Columns[k] = &types.Column{
				DB: newTable.DB,
				Table: newTable.Table,
				Column: k,
				DataType: "func",
				Func: true,
				NewFunc: true,
			}	
		}
	}
	return &newTable
}

func (s *StateFlow) randTable(newName bool, fn bool, online bool) (*types.Table) {
	tables := []*types.Table{}
	for _, table := range s.db.Tables {
		// get online tables
		if !online || !s.db.Online || table.Online {
			tables = append(tables, table)
		}
	}

	if len(tables) == 0 {
		// return nil
		// FIXME: nil not panic
		for _, table := range s.db.Tables {
			tables = append(tables, table)
		}
	}

	return tables[util.Rd(len(tables))].Clone()
}

func (s *StateFlow) randOfflineTable(newName bool, fn bool) (*types.Table) {
	tables := []*types.Table{}
	for _, table := range s.db.Tables {
		// avoid online tables
		if !table.OnlineOther {
			tables = append(tables, table)
		}
	}

	if len(tables) == 0 {
		return nil
	}
	return tables[util.Rd(len(tables))].Clone()
}

func (s *StateFlow) randOriginTable() *types.Table {
	tables := s.db.Tables
	index := 0
	k := util.Rd(len(tables))
	for _, table := range tables {
		if index == k {
			return table.Clone()
		}
		index++
	}
	return nil
}

func (s *StateFlow) randColumns(table *types.Table) []*types.Column {
	var columns []*types.Column
	rate := float64(0.75)
	for _, column := range table.Columns {
		if util.RdFloat64() < rate {
			columns = append(columns, column)
		}
	}
	return columns
}

func (s *StateFlow) mergeTable(table1 *types.Table, table2 *types.Table) (*types.Table, [2]*types.Column) {
	subTableName := s.getSubTableName()
	table := types.Table{
		DB: table1.DB,
		Table: subTableName,
		Type: "SUB TABLE",
		Columns: make(map[string]*types.Column),
	}

	var onColumns [2]*types.Column
	index := 0

	for _, column := range table1.Columns {
		newColumn := types.Column{
			DB: column.DB,
			Table: subTableName,
			OriginTable: column.Table,
			DataType: column.DataType,
			Column: fmt.Sprintf("c%d", index),
			OriginColumn: column.Column,
		}
		if column.NewFunc {
			newColumn.Func = column.Func
		}
		table.Columns[fmt.Sprintf("c%d", index)] = &newColumn
		index++
	}
	for _, column := range table2.Columns {
		newColumn := types.Column{
			DB: column.DB,
			Table: subTableName,
			OriginTable: column.Table,
			DataType: column.DataType,
			Column: fmt.Sprintf("c%d", index),
			OriginColumn: column.Column,
		}
		if column.NewFunc {
			newColumn.Func = column.Func
		}
		table.Columns[fmt.Sprintf("c%d", index)] = &newColumn
		index++
	}

	for _, column1 := range table1.Columns {
		for _, column2 := range table2.Columns {
			if column1.DataType == column2.DataType {
				onColumns[0] = column1
				onColumns[1] = column2
				return &table, onColumns
			}
		}
	}

	return &table, onColumns
}

func (s *StateFlow) renameTable(table *types.Table) *types.Table {
	newName := s.getSubTableName()
	table.OriginTable = table.Table
	table.Table = newName
	for _, column := range table.Columns {
		column.Table = newName
	}
	return table
}

func (s *StateFlow) randNewTable() *types.Table {
	table := types.Table{
		DB: s.db.Name,
		Table: util.RdStringChar(util.RdRange(5, 10)),
		Type: "BASE TABLE",
		Columns: make(map[string]*types.Column),
	}
	table.Columns["id"] = &types.Column{
		DB: table.DB,
		Table: table.Table,
		Column: "id",
		DataType: "int",
		DataLen: 11,
	}
	table.Columns["id"].AddOption(ast.ColumnOptionNotNull)
	table.Columns["id"].AddOption(ast.ColumnOptionAutoIncrement)
	table.Columns["uuid"] = &types.Column{
		DB: table.DB,
		Table: table.Table,
		Column: "uuid",
		DataType: "varchar",
		DataLen: 253,
	}
	columnCount := util.RdRange(4, 20)
	for i := 0; i < columnCount; i++ {
		col := s.randNewColumn()
		col.DB = table.DB
		col.Table = table.Table
		table.Columns[col.Column] = col
	}

	return &table
}

// TIMESTAMP with CURRENT_TIMESTAMP in ALTER TABLE will make difference by design in binlog test
func (s *StateFlow) randNewColumn() *types.Column {
	columnName := util.RdStringChar(util.RdRange(5, 10))
	columnType := util.RdType()
	columnLen := util.RdDataLen(columnType)
	columnOptions := util.RdColumnOptions(columnType)
	if s.stable {
		for i, o := range columnOptions {
			if o == ast.ColumnOptionDefaultValue {
				columnOptions = append(columnOptions[:i], columnOptions[i+1:]...)
			}
		}
	}
	return &types.Column{
		Column: columnName,
		DataType: columnType,
		DataLen: columnLen,
		Options: columnOptions,
	}
}
