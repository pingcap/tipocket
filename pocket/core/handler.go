package core

import (
	"time"
	"math/rand"
	"github.com/pingcap/tipocket/pocket/executor"
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

func (e *Executor) startHandler() {
	for {
		if e.ifLock {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		var (
			err error
			sql = <- e.ch
		)
		time.Sleep(2 * time.Millisecond)

		// Exec SQL
		e.deadlockCh <- sql.executorID
		executor := e.findExecutor(sql.executorID)
		executor.ExecSQL(sql.sql)

		// DDL should wait for execution finish
		switch sql.sql.SQLType {
		case types.SQLTypeTxnBegin, types.SQLTypeTxnCommit, types.SQLTypeTxnRollback,
				 types.SQLTypeDDLCreateTable, types.SQLTypeDDLAlterTable, types.SQLTypeDDLCreateIndex:
			for {
				if len(executor.TxnReadyCh) == 1 {
					break
				}
				time.Sleep(time.Millisecond)
			}
		}

		if err != nil {
			e.logger.Infof("[FAIL] Exec SQL %s error %v", sql.sql.SQLStmt, err)
		} else {
			// e.logger.Infof("[SUCCESS] Exec SQL %s success", sql.SQLStmt)
		}
	}
}

func (e *Executor) randExecutor() *executor.Executor {
	return e.executors[rand.Intn(len(e.executors))]
}

func (e *Executor) randFreeExecutor() *executor.Executor {
	var notInTxns []*executor.Executor
	for _, e := range e.executors {
		if !e.IfTxn() {
			notInTxns = append(notInTxns, e)
		}
	}
	if len(notInTxns) == 0 {
		return nil
	}
	return notInTxns[rand.Intn(len(notInTxns))]
}

func (e *Executor) randBusyExecutor() *executor.Executor {
	var InTxns []*executor.Executor
	for _, e := range e.executors {
		if e.IfTxn() {
			InTxns = append(InTxns, e)
		}
	}
	if len(InTxns) == 0 {
		return nil
	}
	return InTxns[rand.Intn(len(InTxns))]
}

// get free executor if exist, unless get a random executor
func (e *Executor) tryRandFreeExecutor() *executor.Executor {
	if e := e.randFreeExecutor(); e != nil {
		return e
	}
	return e.randExecutor()
}

// get free executor if exist, unless get a random executor
func (e *Executor) tryRandBusyExecutor() *executor.Executor {
	if e := e.randBusyExecutor(); e != nil {
		return e
	}
	return e.randExecutor()
}

// see if can execute DDL(no transactions are in process)
func (e *Executor) canExecuteDDL() bool {
	for _, e := range e.executors {
		if e.IfTxn() {
			return false
		}
	}
	return true
}
