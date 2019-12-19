package core 

import (
	"time"
	"math/rand"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pocket/pkg/types"
	// smith "github.com/pingcap/tipocket/go-sqlsmith"
)

func (e *Executor) smithGenerate() {
	// go syncromous here for sqlsmith require first time init
	e.prepare()
	log.Info("ready to generate")
	for {
		var (
			err error
			rd = rand.Intn(300)
		)
		e.Lock()
		log.Info(e.canExecuteDDL())
		if e.canExecuteDDL() {
			rd = rand.Intn(25)
		} else {
			if rd != 0 {
				rd += 20
			}
		}
		e.Unlock()
		if rd == 0 {
			err = e.generateDDLCreateTable()
		} else if rd < 10 {
			err = e.generateDDLAlterTable()
		} else if rd < 20 {
			err = e.generateDDLCreateIndex()
		} else if rd < 40 {
			e.generateTxnBegin()
		} else if rd < 160 {
			err = e.generateUpdate()
		} else if rd < 180 {
			e.generateTxnCommit()
		} else if rd < 190 {
			e.generateTxnRollback()
		} else {
			// err = e.generateSelect()
			err = e.generateInsert()
		}
		if err != nil {
			log.Fatalf("generate error %v \n", errors.ErrorStack(err))
		}
		time.Sleep(time.Millisecond)
	}
}

func (e *Executor) prepare() {
	for i := 0; i < 10; i++ {
		if err := e.generateDDLCreateTable(); err != nil {
			log.Fatal(err)
		}
	}
	for _, executor := range e.executors {
		executor.ReloadSchema()
	}
}

func (e *Executor) generateDDLCreateTable() error {
	executor := e.randFreeExecutor()
	if executor == nil {
		return nil
	}
	sql, err := executor.GenerateDDLCreateTable()
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &execSQL{
		executorID: executor.GetID(),
		sql: sql,
	}
	return nil
}

func (e *Executor) generateDDLAlterTable() error {
	executor := e.randFreeExecutor()
	if executor == nil {
		return nil
	}
	sql, err := executor.GenerateDDLAlterTable()
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &execSQL{
		executorID: executor.GetID(),
		sql: sql,
	}
	return nil
}

func (e *Executor) generateDDLCreateIndex() error {
	executor := e.randFreeExecutor()
	if executor == nil {
		return nil
	}
	sql, err := executor.GenerateDDLCreateIndex()
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &execSQL{
		executorID: executor.GetID(),
		sql: sql,
	}
	return nil
}

func (e *Executor) generateSelect() error {
	executor := e.tryRandBusyExecutor()
	if executor == nil {
		return nil
	}
	sql, err := executor.GenerateSelect()
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &execSQL{
		executorID: executor.GetID(),
		sql: sql,
	}
	return nil
}

func (e *Executor) generateUpdate() error {
	executor := e.tryRandBusyExecutor()
	if executor == nil {
		return nil
	}
	sql, err := executor.GenerateUpdate()
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &execSQL{
		executorID: executor.GetID(),
		sql: sql,
	}
	return nil
}

func (e *Executor) generateInsert() error {
	executor := e.tryRandBusyExecutor()
	if executor == nil {
		return nil
	}
	sql, err := executor.GenerateInsert()
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &execSQL{
		executorID: executor.GetID(),
		sql: sql,
	}
	return nil	
}

func (e *Executor) generateTxnBegin() {
	executor := e.randFreeExecutor()
	if executor == nil {
		return
	}
	e.ch <- &execSQL{
		executorID: executor.GetID(),
		sql: &types.SQL{
			SQLType: types.SQLTypeTxnBegin,
			SQLStmt: "BEGIN",
		},
	}
}

func (e *Executor) generateTxnCommit() {
	executor := e.randBusyExecutor()
	if executor == nil {
		return
	}
	e.ch <- &execSQL{
		executorID: executor.GetID(),
		sql: &types.SQL{
			SQLType: types.SQLTypeTxnCommit,
			SQLStmt: "COMMIT",
		},
	}
}

func (e *Executor) generateTxnRollback() {
	executor := e.randBusyExecutor()
	if executor == nil {
		return
	}
	e.ch <- &execSQL{
		executorID: executor.GetID(),
		sql: &types.SQL{
			SQLType: types.SQLTypeTxnRollback,
			SQLStmt: "ROLLBACK",
		},
	}
}
