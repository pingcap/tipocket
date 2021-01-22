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

package executor

import (
	"fmt"
	"strings"

	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/generator/generator"
	smith "github.com/pingcap/tipocket/testcase/pocket/pkg/go-sqlsmith"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/types"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/util"
)

// ReloadSchema expose reloadSchema
func (e *Executor) ReloadSchema() error {
	return errors.Trace(e.reloadSchema())
}

func (e *Executor) reloadSchema() error {
	schema, err := e.conn1.FetchSchema(e.dbname)
	if err != nil {
		return errors.Trace(err)
	}
	indexes := make(map[string][]string)
	for _, col := range schema {
		if _, ok := indexes[col[2]]; ok {
			continue
		}
		index, err := e.conn1.FetchIndexes(e.dbname, col[1])
		// may not return error here
		// just disable indexes
		if err != nil {
			return errors.Trace(err)
		}
		indexes[col[1]] = index
	}

	e.ss = smith.New()
	e.ss.LoadSchema(schema, indexes)
	e.ss.SetDB(e.dbname)
	e.ss.SetStable(e.opt.Stable)
	e.ss.SetHint(e.opt.Hint)
	e.BeginWithOnlineTables()
	return nil
}

// Generate DDL

// GenerateDDLCreateTable rand create table statement
func (e *Executor) GenerateDDLCreateTable() (*types.SQL, error) {
	stmt, table, err := e.ss.CreateTableStmt()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &types.SQL{
		SQLType:  types.SQLTypeDDLCreateTable,
		SQLTable: table,
		SQLStmt:  stmt,
	}, nil
}

// GenerateDDLCreateIndex rand create index statement
func (e *Executor) GenerateDDLCreateIndex(opt *generator.DDLOptions) (*types.SQL, error) {
	stmt, err := e.ss.CreateIndexStmt(opt)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &types.SQL{
		SQLType: types.SQLTypeDDLCreateIndex,
		SQLStmt: stmt,
	}, nil
}

// GenerateDDLAlterTable rand alter table statement
func (e *Executor) GenerateDDLAlterTable(opt *generator.DDLOptions) (*types.SQL, error) {
	stmt, err := e.ss.AlterTableStmt(opt)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &types.SQL{
		SQLType: types.SQLTypeDDLAlterTable,
		SQLStmt: stmt,
	}, nil
}

// GenerateDMLSelect rand select statement
func (e *Executor) GenerateDMLSelect() (*types.SQL, error) {
	stmt, table, err := e.ss.SelectStmt(4)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &types.SQL{
		SQLType:  types.SQLTypeDMLSelect,
		SQLStmt:  stmt,
		SQLTable: table,
	}, nil
}

// GenerateDMLSelectForUpdate rand update statement
func (e *Executor) GenerateDMLSelectForUpdate() (*types.SQL, error) {
	stmt, table, err := e.ss.SelectForUpdateStmt(4)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &types.SQL{
		SQLType:  types.SQLTypeDMLSelectForUpdate,
		SQLStmt:  stmt,
		SQLTable: table,
	}, nil
}

// GenerateDMLUpdate rand update statement
func (e *Executor) GenerateDMLUpdate() (*types.SQL, error) {
	stmt, table, err := e.ss.UpdateStmt()
	if err != nil {
		return nil, errors.Trace(err)
	}
	if strings.HasPrefix(stmt, "UPDATE s0") {
		log.Info(stmt, table)
		log.Info(e.conn1.FetchSchema(e.dbname))
	}
	return &types.SQL{
		SQLType:  types.SQLTypeDMLUpdate,
		SQLStmt:  stmt,
		SQLTable: table,
	}, nil
}

// GenerateDMLDelete rand update statement
func (e *Executor) GenerateDMLDelete() (*types.SQL, error) {
	stmt, table, err := e.ss.DeleteStmt()
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &types.SQL{
		SQLType:  types.SQLTypeDMLDelete,
		SQLStmt:  stmt,
		SQLTable: table,
	}, nil
}

// GenerateDMLInsert rand insert statement
func (e *Executor) GenerateDMLInsert() (*types.SQL, error) {
	stmt, table, err := e.ss.InsertStmt(false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &types.SQL{
		SQLType:  types.SQLTypeDMLInsert,
		SQLStmt:  stmt,
		SQLTable: table,
	}, nil
}

// GenerateSleep rand insert statement
func (e *Executor) GenerateSleep() *types.SQL {
	duration := util.Rd(25)
	return &types.SQL{
		SQLType:  types.SQLTypeSleep,
		SQLStmt:  fmt.Sprintf("SELECT SLEEP(%d)", duration),
		ExecTime: duration,
	}
}

// BeginWithOnlineTables begins transaction with online tables
func (e *Executor) BeginWithOnlineTables() {
	e.OnlineTable = e.ss.BeginWithOnlineTables(&generator.DMLOptions{
		OnlineTable: e.opt.OnlineDDL,
	})
}

// EndTransaction clear online tables
func (e *Executor) EndTransaction() {
	e.OnlineTable = e.ss.EndTransaction()
}

// GenerateTxnBegin start transaction
func (e *Executor) GenerateTxnBegin() *types.SQL {
	return &types.SQL{
		SQLType: types.SQLTypeTxnBegin,
		SQLStmt: "BEGIN",
	}
}

// GenerateTxnCommit commit transaction
func (e *Executor) GenerateTxnCommit() *types.SQL {
	return &types.SQL{
		SQLType: types.SQLTypeTxnCommit,
		SQLStmt: "COMMIT",
	}
}

// GenerateTxnRollback rollback transaction
func (e *Executor) GenerateTxnRollback() *types.SQL {
	return &types.SQL{
		SQLType: types.SQLTypeTxnRollback,
		SQLStmt: "ROLLBACK",
	}
}
