// Copyright 2020 PingCAP, Inc.
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
	"sync"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/pocket/pkg/types"
	"github.com/pingcap/tipocket/pkg/pocket/util"
)

func (e *Executor) dmTest() {
	for {
		e.ErrCh <- e.execDMTestSQL(<-e.SQLCh)
	}
}

func (e *Executor) execDMTestSQL(sql *types.SQL) error {
	e.logStmtTodo(sql.SQLStmt)
	var err error
	switch sql.SQLType {
	case types.SQLTypeDMLInsert, types.SQLTypeDMLUpdate, types.SQLTypeDMLDelete,
		types.SQLTypeTxnBegin, types.SQLTypeTxnCommit, types.SQLTypeTxnRollback,
		types.SQLTypeCreateDatabase, types.SQLTypeDropDatabase,
		types.SQLTypeDDLCreateTable, types.SQLTypeDDLAlterTable,
		types.SQLTypeDDLCreateIndex,
		types.SQLTypeExec, types.SQLTypeSleep:
		err = e.execDMTestStmt(sql.SQLStmt)
	case types.SQLTypeExit:
		e.Stop("receive exit SQL signal")
	default:
		log.Debugf("ignore SQL %v", sql)
	}

	e.logStmtResult(sql.SQLStmt, err)
	return err
}

func (e *Executor) execDMTestStmt(sql string) error {
	var (
		wg   sync.WaitGroup
		err1 error
		err2 error
	)

	wg.Add(2)
	go func() {
		err1 = e.conn1.Exec(sql)
		wg.Done()
	}()
	go func() {
		// TODO(csuzhangxc): shard merge support
		err2 = e.conn2.Exec(sql)
		wg.Done()
	}()
	wg.Wait()

	if err := util.ErrorMustSame(err1, err2); err != nil {
		return err
	}
	return err1
}

func (e *Executor) dmTestIfTxn() bool {
	return e.conn1.IfTxn() || e.conn2.IfTxn()
}
