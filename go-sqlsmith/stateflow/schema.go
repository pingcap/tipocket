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

func (s *StateFlow) randTable(newName bool, fn bool) (*types.Table) {
	newTable := new(types.Table)
	tables := s.db.Tables
	index := 0
	k := util.Rd(len(tables))
	for _, table := range tables {
		if index == k {
			for len(newTable.Columns) == 0 {
				newTable = s.randTableFromTable(table, newName, fn)
			}
			return newTable
		}
		index++
	}
	// should not reach here
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
	columnCount := util.RdRange(4, 20)
	for i := 0; i < columnCount; i++ {
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
		table.Columns[columnName] = &types.Column{
			DB: table.DB,
			Table: table.Table,
			Column: columnName,
			DataType: columnType,
			DataLen: columnLen,
			Options: columnOptions,
		}
	}

	return &table
}
