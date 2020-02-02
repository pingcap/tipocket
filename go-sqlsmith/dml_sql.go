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
	"errors"
	"fmt"
	"github.com/pingcap/tipocket/go-sqlsmith/builtin"
	"github.com/pingcap/tipocket/go-sqlsmith/types"
	"github.com/pingcap/tipocket/go-sqlsmith/util"
	"strings"
)

// SelectStmt make random select statement SQL
func (s *SQLSmith) SelectStmt(depth int) (string, string, error) {
	tree := s.selectStmt(depth)
	return s.Walk(tree)
}

// SelectForUpdateStmt make random select statement SQL with for update lock
func (s *SQLSmith) SelectForUpdateStmt(depth int) (string, string, error) {
	tree := s.selectForUpdateStmt(depth)
	return s.Walk(tree)
}

// UpdateStmt make random update statement SQL
func (s *SQLSmith) UpdateStmt() (string, string, error) {
	tree := s.updateStmt()
	return s.Walk(tree)
}

// InsertStmt implement insert statement from AST
func (s *SQLSmith) InsertStmt(fn bool) (string, string, error) {
	tree := s.insertStmt()
	return s.Walk(tree)
}

// DeleteStmt implement delete statement from AST
func (s *SQLSmith) DeleteStmt() (string, string, error) {
	tree := s.deleteStmt()
	return s.Walk(tree)
}

// InsertStmtStr make random insert statement SQL
func (s *SQLSmith) InsertStmtStr(fn bool) (string, string, error) {
	if s.currDB == "" {
		return "", "", errors.New("no table selected")
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
				return "", "", err
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
	return sql, table.Table, nil
}
