package sqlsmith

import (
	"errors"
	"fmt"
	"strings"
	"github.com/pingcap/tipocket/go-sqlsmith/builtin"
	"github.com/pingcap/tipocket/go-sqlsmith/types"
	"github.com/pingcap/tipocket/go-sqlsmith/util"
)

// SelectStmt make random select statement SQL
func (s *SQLSmith) SelectStmt(depth int) (string, error) {
	tree := s.selectStmt(depth)
	return s.Walk(tree)
}

// UpdateStmt make random update statement SQL
func (s *SQLSmith) UpdateStmt() (string, error) {
	tree := s.updateStmt()
	return s.Walk(tree)
}

// InsertStmtAST implement insert statement from AST
func (s *SQLSmith) InsertStmtAST() (string, error) {
	tree := s.insertStmt()
	return s.Walk(tree)
}

// InsertStmt make random insert statement SQL
func (s *SQLSmith) InsertStmt(fn bool) (string, error) {
	if s.currDB == "" {
		return "", errors.New("no table selected")
	}
	var table *types.Table
	rdTableIndex := s.rd(len(s.Databases[s.currDB].Tables))
	tableIndex := 0
	for _, t := range s.Databases[s.currDB].Tables {
		if rdTableIndex == tableIndex {
			table = t
			break
		}
		tableIndex++
	}

	var columns []*types.Column
	var columnNames []string
	for _, column := range table.Columns {
		if column.Column == "id" {
			continue
		}
		columns = append(columns, column)
		columnNames = append(columnNames, fmt.Sprintf("`%s`", column.Column))
	}

	var vals []string
	for _, column := range columns {
		if fn && s.rdFloat64() < 0.5 {
			builtinFn := builtin.GenerateFuncCallExpr(nil, s.rd(3), s.stable)
			builtinStr, err := util.BufferOut(builtinFn)
			if err != nil {
				return "", err
			}
			vals = append(vals, builtinStr)
		} else {
			vals = append(vals, s.generateDataItem(column.DataType))
		}
	}
	sql := fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s)",
		table.Table,
		strings.Join(columnNames, ", "),
		strings.Join(vals, ", "))
	return sql, nil
}
