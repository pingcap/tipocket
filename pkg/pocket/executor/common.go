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
)

// Select offer unified method for single & abtest
func (e *Executor) Select(stmt string) error {
	e.Lock()
	defer e.Unlock()
	switch e.mode {
	case "abtest":
		return e.ABTestSelect(stmt)
	case "single":
		return e.SingleTestSelect(stmt)
	}
	panic("unhandled select switch")
}

// Insert offer unified method for single & abtest
func (e *Executor) Insert(stmt string) error {
	e.Lock()
	defer e.Unlock()
	switch e.mode {
	case "abtest":
		return e.ABTestInsert(stmt)
	case "single":
		return e.SingleTestInsert(stmt)
	}
	panic("unhandled select switch")
}

// Update offer unified method for single & abtest
func (e *Executor) Update(stmt string) error {
	e.Lock()
	defer e.Unlock()
	switch e.mode {
	case "abtest":
		return e.ABTestUpdate(stmt)
	case "single":
		return e.SingleTestUpdate(stmt)
	}
	panic("unhandled select switch")
}

// Delete offer unified method for single & abtest
func (e *Executor) Delete(stmt string) error {
	e.Lock()
	defer e.Unlock()
	switch e.mode {
	case "abtest":
		return e.ABTestDelete(stmt)
	case "single":
		return e.SingleTestDelete(stmt)
	}
	panic("unhandled select switch")
}

// ExecDDL offer unified method for single & abtest
func (e *Executor) ExecDDL(stmt string) error {
	e.Lock()
	defer e.Unlock()
	switch e.mode {
	case "abtest":
		return e.ABTestExecDDL(stmt)
	case "single":
		return e.SingleTestExecDDL(stmt)
	}
	panic("unhandled select switch")
}

// TxnBegin offer unified method for single & abtest
func (e *Executor) TxnBegin() error {
	e.Lock()
	defer e.Unlock()
	switch e.mode {
	case "abtest":
		return e.ABTestTxnBegin()
	case "single":
		return e.SingleTestTxnBegin()
	}
	panic("unhandled txn begin switch")
}

// TxnCommit offer unified method for single & abtest
func (e *Executor) TxnCommit() error {
	e.Lock()
	defer e.Unlock()
	switch e.mode {
	case "abtest":
		return e.ABTestTxnCommit()
	case "single":
		return e.SingleTestTxnCommit()
	}
	panic("unhandled txn commit switch")
}

// TxnRollback offer unified method for single & abtest
func (e *Executor) TxnRollback() error {
	e.Lock()
	defer e.Unlock()
	switch e.mode {
	case "abtest":
		return e.ABTestTxnRollback()
	case "single":
		return e.SingleTestTxnRollback()
	}
	panic("unhandled txn rollback switch")
}

// since the tiflash cluster is always conn1 in TiFlash abtest, so it is
// okay to just use single test ddl here.
func (e *Executor) createTiFlashTableReplica(table string) error {
	return e.SingleTestExecDDL(fmt.Sprintf("ALTER TABLE %s SET TIFLASH REPLICA 1", table))
}

func (e *Executor) waitTiFlashTableSync(table string) error {
	sql := fmt.Sprintf("SELECT AVAILABLE FROM information_schema.tiflash_replica WHERE TABLE_SCHEMA = '%s' and TABLE_NAME = '%s'",
		e.dbname, table)
	for {
		res, err := e.GetConn().Select(sql)
		if err != nil {
			return err
		}
		// table doesn't exist
		if res == nil {
			return nil
		}
		// table sync completed
		if res[0][0].ValString == "1" {
			break
		}
	}
	return nil
}
