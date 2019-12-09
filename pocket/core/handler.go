package core

import (
	"time"
	"math/rand"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pocket/executor"
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

func (e *Executor) startHandler() {
	// switch e.mode {
	// case "single":
	// 	e.singleTest()
	// case "abtest":
	// 	e.abTest()
	// case "binlog":
	// 	e.singleTest()
	// default:
	// 	panic("unhandled test mode")
	// }
	for {
		if e.coreOpt.Reproduce == "" {
			time.Sleep(2*time.Millisecond)
		}
		var (
			err error
			sql = <- e.ch
		)

		// Exec SQL
		switch sql.SQLType {
		case types.SQLTypeReloadSchema:
			err = e.reloadSchema()
			if err != nil {
				log.Fatalf("reload schema failed %+v\n", errors.ErrorStack(err))
			}
		case types.SQLTypeExit:
			e.Stop("receive exit SQL signal")
		case types.SQLTypeTxnBegin:
			executor := e.randFreeExecutor()
			if executor != nil {
				e.deadlockCh <- executor.GetID()
				executor.ExecSQL(sql)
			}
		case types.SQLTypeTxnCommit, types.SQLTypeTxnRollback:
			executor := e.randBusyExecutor()
			if executor != nil {
				e.deadlockCh <- executor.GetID()
				executor.ExecSQL(sql)
			}
		case types.SQLTypeDDLCreate:
			executor := e.tryRandFreeExecutor()
			e.deadlockCh <- executor.GetID()
			executor.ExecSQL(sql)
		default:
			executor := e.randBusyExecutor()
			if executor != nil {
				e.deadlockCh <- executor.GetID()
				executor.ExecSQL(sql)
			}
		}

		if err != nil {
			e.logger.Infof("[FAIL] Exec SQL %s error %v", sql.SQLStmt, err)
		} else {
			// e.logger.Infof("[SUCCESS] Exec SQL %s success", sql.SQLStmt)
		}
	}
}

func (e *Executor) singleTest() {
	for {
		var (
			err error
			sql = <- e.ch
		)

		// Exec SQL
		switch sql.SQLType {
		case types.SQLTypeReloadSchema:
			err = e.reloadSchema()
			if err != nil {
				log.Fatalf("reload schema failed %+v\n", errors.ErrorStack(err))
			}
		case types.SQLTypeExit:
			e.Stop("receive exit SQL signal")
		case types.SQLTypeTxnBegin:
			executor := e.randFreeExecutor()
			if executor != nil {
				e.deadlockCh <- executor.GetID()
				executor.ExecSQL(sql)
				<-executor.TxnReadyCh
			}
		case types.SQLTypeTxnCommit, types.SQLTypeTxnRollback:
			executor := e.randBusyExecutor()
			if executor != nil {
				e.deadlockCh <- executor.GetID()
				executor.ExecSQL(sql)
				<-executor.TxnReadyCh
			}
		case types.SQLTypeDDLCreate:
			executor := e.tryRandFreeExecutor()
			e.deadlockCh <- executor.GetID()
			executor.ExecSQL(sql)
			<-executor.TxnReadyCh
			if executor.IfTxn() {
				executor.ExecSQL(&types.SQL{
					SQLType: types.SQLTypeTxnCommit,
					SQLStmt: "COMMIT",
				})
				<-executor.TxnReadyCh
			}
		default:
			executor := e.randBusyExecutor()
			if executor != nil {
				e.deadlockCh <- executor.GetID()
				executor.ExecSQL(sql)
			}
		}

		if err != nil {
			e.logger.Infof("[FAIL] Exec SQL %s error %v", sql.SQLStmt, err)
		} else {
			// e.logger.Infof("[SUCCESS] Exec SQL %s success", sql.SQLStmt)
		}
	}
}

func (e *Executor) abTest() {
	for {
		var (
			err error
			sql = <- e.ch
		)

		// Exec SQL
		switch sql.SQLType {
		case types.SQLTypeReloadSchema:
			err = e.reloadSchema()
			if err != nil {
				log.Fatalf("reload schema failed %+v\n", errors.ErrorStack(err))
			}
		case types.SQLTypeExit:
			e.Stop("receive exit SQL signal")
		case types.SQLTypeTxnBegin:
			executor := e.randFreeExecutor()
			if executor != nil {
				e.deadlockCh <- executor.GetID()
				executor.ExecSQL(sql)
				<-executor.TxnReadyCh
			}
		case types.SQLTypeTxnCommit, types.SQLTypeTxnRollback:
			executor := e.randBusyExecutor()
			if executor != nil {
				e.deadlockCh <- executor.GetID()
				executor.ExecSQL(sql)
				<-executor.TxnReadyCh
			}
		case types.SQLTypeDDLCreate:
			executor := e.tryRandFreeExecutor()
			e.deadlockCh <- executor.GetID()
			executor.ExecSQL(sql)
			<-executor.TxnReadyCh
		default:
			executor := e.randBusyExecutor()
			if executor != nil {
				e.deadlockCh <- executor.GetID()
				executor.ExecSQL(sql)
			}
		}

		if err != nil {
			e.logger.Infof("[FAIL] Exec SQL %s error %v", sql.SQLStmt, err)
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
