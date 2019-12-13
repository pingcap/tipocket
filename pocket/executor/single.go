package executor

import (
	"fmt"
	"github.com/juju/errors"
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

func (e *Executor) singleTest() {
	for {
		var (
			err error
			sql = <- e.ch
		)
		e.Lock()
		e.logStmtTodo(sql.SQLStmt)

		switch sql.SQLType {
		case types.SQLTypeDMLSelect:
			err = e.singleTestSelect(sql.SQLStmt)
		case types.SQLTypeDMLUpdate:
			err = e.singleTestUpdate(sql.SQLStmt)
		case types.SQLTypeDMLInsert:
			err = e.singleTestInsert(sql.SQLStmt)
		case types.SQLTypeDMLDelete:
			err = e.singleTestDelete(sql.SQLStmt)
		case types.SQLTypeDDLCreate:
			err = e.singleTestCreateTable(sql.SQLStmt)
		case types.SQLTypeTxnBegin:
			err = e.singleTestTxnBegin()
		case types.SQLTypeTxnCommit:
			err = e.singleTestTxnCommit()
		case types.SQLTypeTxnRollback:
			err = e.singleTestTxnRollback()
		case types.SQLTypeExec:
			e.singleTestExec(sql.SQLStmt)
		case types.SQLTypeExit:
			e.Stop("receive exit SQL signal")
		default:
			panic(fmt.Sprintf("unhandled case %+v", sql))
		}

		e.logStmtResult(sql.SQLStmt, err)
		e.Unlock()
	}
}

// SingleTestSelect expose singleTestSelect
func (e *Executor) SingleTestSelect(sql string) error {
	e.logStmtTodo(sql)
	err := e.singleTestSelect(sql)
	e.logStmtResult(sql, err)
	<- e.TxnReadyCh
	return err
}

// SingleTestInsert expose singleTestInsert
func (e *Executor) SingleTestInsert(sql string) error {
	e.logStmtTodo(sql)
	err := e.singleTestInsert(sql)
	e.logStmtResult(sql, err)
	<- e.TxnReadyCh
	return err
}

// SingleTestUpdate expose singleTestUpdate
func (e *Executor) SingleTestUpdate(sql string) error {
	e.logStmtTodo(sql)
	err := e.singleTestUpdate(sql)
	e.logStmtResult(sql, err)
	<- e.TxnReadyCh
	return err
}

// SingleTestDelete expose singleTestDelete
func (e *Executor) SingleTestDelete(sql string) error {
	e.logStmtTodo(sql)
	err := e.singleTestDelete(sql)
	e.logStmtResult(sql, err)
	<- e.TxnReadyCh
	return err
}

// SingleTestCreateTable expose singleTestCreateTable
func (e *Executor) SingleTestCreateTable(sql string) error {
	e.logStmtTodo(sql)
	err := e.singleTestCreateTable(sql)
	e.logStmtResult(sql, err)
	<- e.TxnReadyCh
	return err
}

// SingleTestTxnBegin export singleTestTxnBegin
func (e *Executor) SingleTestTxnBegin() error {
	e.logStmtTodo("BEGIN")
	err := e.singleTestTxnBegin()
	e.logStmtResult("BEGIN", err)
	<- e.TxnReadyCh
	return err
}

// SingleTestTxnCommit export singleTestTxnCommit
func (e *Executor) SingleTestTxnCommit() error {
	e.logStmtTodo("COMMIT")
	err := e.singleTestTxnCommit()
	e.logStmtResult("COMMIT", err)
	<- e.TxnReadyCh
	return err
}

// SingleTestTxnRollback export singleTestTxnRollback
func (e *Executor) SingleTestTxnRollback() error {
	e.logStmtTodo("ROLLBACK")
	err := e.singleTestTxnRollback()
	e.logStmtResult("ROLLBACK", err)
	<- e.TxnReadyCh
	return err
}

// SingleTestIfTxn expose singleTestIfTxn
func (e *Executor) SingleTestIfTxn() bool {
	return e.singleTestIfTxn()
}

// DML
func (e *Executor) singleTestSelect(sql string) error {
	_, err := e.conn1.Select(sql)
	e.TxnReadyCh <- struct{}{}
	return errors.Trace(err)
}

func (e *Executor) singleTestUpdate(sql string) error {
	err := e.conn1.Update(sql)
	e.TxnReadyCh <- struct{}{}
	return errors.Trace(err)
}

func (e *Executor) singleTestInsert(sql string) error {
	err := e.conn1.Insert(sql)
	e.TxnReadyCh <- struct{}{}
	return errors.Trace(err)
}

func (e *Executor) singleTestDelete(sql string) error {
	err := e.conn1.Delete(sql)
	e.TxnReadyCh <- struct{}{}
	return errors.Trace(err)
}

// DDL
func (e *Executor) singleTestCreateTable(sql string) error {
	err := e.conn1.ExecDDL(sql)
	// continue generate
	e.TxnReadyCh <- struct{}{}
	return errors.Trace(err)
}


// just execute
func (e *Executor) singleTestExec(sql string) {
	_ = e.conn1.Exec(sql)
}

func (e *Executor) singleTestTxnBegin() error {
	err := e.conn1.Begin()
	// continue generate
	e.TxnReadyCh <- struct{}{}
	return errors.Trace(err)
}

func (e *Executor) singleTestTxnCommit() error {
	err := e.conn1.Commit()
	// continue generate
	e.TxnReadyCh <- struct{}{}
	return errors.Trace(err)
}

func (e *Executor) singleTestTxnRollback() error {
	err := e.conn1.Rollback()
	// continue generate
	e.TxnReadyCh <- struct{}{}
	return errors.Trace(err)
}

func (e *Executor) singleTestIfTxn() bool {
	return e.conn1.IfTxn()
}
