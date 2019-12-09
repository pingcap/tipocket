package core 

import (
	"math/rand"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/abclient/pkg/types"
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
		// rd = 100
		if rd == 0 {
			err = e.generateDDLCreate()
		} else if rd < 80 {
			err = e.generateInsert()
		} else if rd < 160 {
			err = e.generateUpdate()
		} else if rd < 170 {
			e.generateTxnBegin()
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
	}
}

func (e *Executor) prepare() {
	for i := 0; i < 10; i++ {
		if err := e.generateDDLCreate(); err != nil {
			log.Fatal(err)
		}
	}
}

func (e *Executor) generateDDLCreate() error {
	stmt, err := e.ss.CreateTableStmt()
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeDDLCreate,
		SQLStmt: stmt,
	}
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeReloadSchema,
	}
	return nil
}

func (e *Executor) generateSelect() error {
	stmt, err := e.ss.SelectStmt(1)
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeDMLSelect,
		SQLStmt: stmt,
	}
	return nil
}

func (e *Executor) generateUpdate() error {
	stmt, err := e.ss.UpdateStmt()
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeDMLUpdate,
		SQLStmt: stmt,
	}
	return nil	
}

func (e *Executor) generateInsert() error {
	stmt, err := e.ss.InsertStmtAST()
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeDMLInsert,
		SQLStmt: stmt,
	}
	return nil	
}

func (e *Executor) generateTxnBegin() {
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeTxnBegin,
		SQLStmt: "BEGIN",
	}
}

func (e *Executor) generateTxnCommit() {
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeTxnCommit,
		SQLStmt: "COMMIT",
	}
}

func (e *Executor) generateTxnRollback() {
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeTxnRollback,
		SQLStmt: "ROLLBACK",
	}
}
