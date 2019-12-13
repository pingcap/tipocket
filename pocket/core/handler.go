package core

import (
	"time"
	"math/rand"
	"github.com/pingcap/tipocket/pocket/executor"
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
		e.findExecutor(sql.executorID).ExecSQL(sql.sql)

		if err != nil {
			e.logger.Infof("[FAIL] Exec SQL %s error %v", sql.sql.SQLStmt, err)
		} else {
			// e.logger.Infof("[SUCCESS] Exec SQL %s success", sql.SQLStmt)
		}
	}
}

func (e *Executor) randExecutor() *executor.Executor {
	e.Lock()
	defer e.Unlock()
	return e.executors[rand.Intn(len(e.executors))]
}

func (e *Executor) randFreeExecutor() *executor.Executor {
	e.Lock()
	defer e.Unlock()
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
	e.Lock()
	defer e.Unlock()
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
