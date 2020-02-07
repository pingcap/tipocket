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
	"sync"

	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/pocket/connection"
	"github.com/pingcap/tipocket/pkg/pocket/pkg/types"
	"github.com/pingcap/tipocket/pkg/pocket/util"
)

func (e *Executor) abTest() {
	for {
		e.ErrCh <- e.execABTestSQL(<-e.SQLCh)
	}
}

func (e *Executor) execABTestSQL(sql *types.SQL) error {
	var err error
	e.logStmtTodo(sql.SQLStmt)

	switch sql.SQLType {
	case types.SQLTypeDMLSelect, types.SQLTypeDMLSelectForUpdate:
		err = e.abTestSelect(sql.SQLStmt)
	case types.SQLTypeDMLUpdate:
		err = e.abTestUpdate(sql.SQLStmt)
	case types.SQLTypeDMLInsert:
		err = e.abTestInsert(sql.SQLStmt)
	case types.SQLTypeDMLDelete:
		err = e.abTestDelete(sql.SQLStmt)
	case types.SQLTypeDDLCreateTable, types.SQLTypeDDLAlterTable, types.SQLTypeDDLCreateIndex:
		err = e.abTestExecDDL(sql.SQLStmt)
	case types.SQLTypeTxnBegin:
		if err := e.reloadSchema(); err != nil {
			log.Error(err)
		}
		err = e.abTestTxnBegin()
	case types.SQLTypeTxnCommit:
		err = e.abTestTxnCommit()
	case types.SQLTypeTxnRollback:
		err = e.abTestTxnRollback()
	case types.SQLTypeExec, types.SQLTypeSleep:
		e.abTestExec(sql.SQLStmt)
	case types.SQLTypeExit:
		e.Stop("receive exit SQL signal")
	default:
		panic(fmt.Sprintf("unhandled case %+v", sql))
	}

	e.logStmtResult(sql.SQLStmt, err)

	return err
}

// ABTestSelect expose abTestSelect
func (e *Executor) ABTestSelect(sql string) error {
	e.logStmtTodo(sql)
	err := e.abTestSelect(sql)
	e.logStmtResult(sql, err)
	return err
}

// ABTestInsert expose abTestInsert
func (e *Executor) ABTestInsert(sql string) error {
	e.logStmtTodo(sql)
	err := e.abTestInsert(sql)
	e.logStmtResult(sql, err)
	return err
}

// ABTestUpdate expose abTestUpdate
func (e *Executor) ABTestUpdate(sql string) error {
	e.logStmtTodo(sql)
	err := e.abTestUpdate(sql)
	e.logStmtResult(sql, err)
	return err
}

// ABTestDelete expose abTestDelete
func (e *Executor) ABTestDelete(sql string) error {
	e.logStmtTodo(sql)
	err := e.abTestDelete(sql)
	e.logStmtResult(sql, err)
	return err
}

// ABTestExecDDL expose abTestExecDDL
func (e *Executor) ABTestExecDDL(sql string) error {
	e.logStmtTodo(sql)
	err := e.abTestExecDDL(sql)
	e.logStmtResult(sql, err)
	return err
}

// ABTestTxnBegin export abTestTxnBegin
func (e *Executor) ABTestTxnBegin() error {
	e.logStmtTodo("BEGIN")
	err := e.abTestTxnBegin()
	e.logStmtResult("BEGIN", err)
	return err
}

// ABTestTxnCommit export abTestTxnCommit
func (e *Executor) ABTestTxnCommit() error {
	e.logStmtTodo("COMMIT")
	err := e.abTestTxnCommit()
	e.logStmtResult("COMMIT", err)
	return err
}

// ABTestTxnRollback export abTestTxnRollback
func (e *Executor) ABTestTxnRollback() error {
	e.logStmtTodo("ROLLBACK")
	err := e.abTestTxnRollback()
	e.logStmtResult("ROLLBACK", err)
	return err
}

// ABTestIfTxn expose abTestIfTxn
func (e *Executor) ABTestIfTxn() bool {
	return e.abTestIfTxn()
}

// DML
func (e *Executor) abTestSelect(sql string) error {
	var (
		wg   sync.WaitGroup
		res1 [][]*connection.QueryItem
		res2 [][]*connection.QueryItem
		err1 error
		err2 error
	)
	wg.Add(2)
	go func() {
		res1, err1 = e.conn1.Select(sql)
		wg.Done()
	}()
	go func() {
		res2, err2 = e.conn2.Select(sql)
		wg.Done()
	}()
	wg.Wait()

	// log.Info("select abtest err", err1, err2)
	if err := util.ErrorMustSame(err1, err2); err != nil {
		return err
	}

	if len(res1) != len(res2) {
		return errors.Errorf("row number not match res1: %d, res2: %d", len(res1), len(res2))
	}
	for index := range res1 {
		var (
			row1 = res1[index]
			row2 = res2[index]
		)

		if len(row1) != len(row1) {
			return errors.Errorf("column number not match res1: %d, res2: %d", len(res1), len(res2))
		}

		for rIndex := range row1 {
			var (
				item1 = row1[rIndex]
				item2 = row2[rIndex]
			)
			if err := item1.MustSame(item2); err != nil {
				return errors.Errorf("%s, row index %d, column index %d", err.Error(), index, rIndex)
			}
		}
	}

	return nil
}

func (e *Executor) abTestUpdate(sql string) error {
	var (
		wg   sync.WaitGroup
		err1 error
		err2 error
	)
	wg.Add(2)
	go func() {
		err1 = e.conn1.Update(sql)
		wg.Done()
	}()
	go func() {
		err2 = e.conn2.Update(sql)
		wg.Done()
	}()
	wg.Wait()

	return util.ErrorMustSame(err1, err2)
}

func (e *Executor) abTestInsert(sql string) error {
	var (
		wg   sync.WaitGroup
		err1 error
		err2 error
	)
	wg.Add(2)
	go func() {
		err1 = e.conn1.Insert(sql)
		wg.Done()
	}()
	go func() {
		err2 = e.conn2.Insert(sql)
		wg.Done()
	}()
	wg.Wait()

	return util.ErrorMustSame(err1, err2)
}

func (e *Executor) abTestDelete(sql string) error {
	var (
		wg   sync.WaitGroup
		err1 error
		err2 error
	)
	wg.Add(2)
	go func() {
		err1 = e.conn1.Delete(sql)
		wg.Done()
	}()
	go func() {
		err2 = e.conn2.Delete(sql)
		wg.Done()
	}()
	wg.Wait()
	return util.ErrorMustSame(err1, err2)
}

// DDL
func (e *Executor) abTestExecDDL(sql string) error {
	var (
		wg   sync.WaitGroup
		err1 error
		err2 error
	)
	wg.Add(2)
	go func() {
		err1 = e.conn1.ExecDDL(sql)
		_ = e.conn1.Commit()
		wg.Done()
	}()
	go func() {
		err2 = e.conn2.ExecDDL(sql)
		_ = e.conn2.Commit()
		wg.Done()
	}()
	wg.Wait()
	return util.ErrorMustSame(err1, err2)
}

// just execute
func (e *Executor) abTestExec(sql string) {
	var (
		wg sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		_ = e.conn1.Exec(sql)
		wg.Done()
	}()
	go func() {
		_ = e.conn2.Exec(sql)
		wg.Done()
	}()
	wg.Wait()
}

func (e *Executor) abTestTxnBegin() error {
	var (
		err1 error
		err2 error
	)
	err1 = e.conn1.Begin()
	err2 = e.conn2.Begin()
	// continue generate
	return util.ErrorMustSame(err1, err2)
}

func (e *Executor) abTestTxnCommit() error {
	var (
		err1 error
		err2 error
	)
	err1 = e.conn1.Commit()
	err2 = e.conn2.Commit()
	// continue generate
	return util.ErrorMustSame(err1, err2)
}

func (e *Executor) abTestTxnRollback() error {
	var (
		err1 error
		err2 error
	)
	err1 = e.conn1.Rollback()
	err2 = e.conn2.Rollback()
	// continue generate
	return util.ErrorMustSame(err1, err2)
}

func (e *Executor) abTestIfTxn() bool {
	return e.conn1.IfTxn() || e.conn2.IfTxn()
}
