package executor

import (
	"math/rand"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pocket/pkg/types"
	// smith "github.com/pingcap/tipocket/go-sqlsmith"
)

func (e *Executor) smithGenerate() {
	e.prepare()
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeReloadSchema,
	}
	log.Info("ready to generate")
	for {
		var (
			err error
			rd = rand.Intn(100)
		)
		// rd = 100
		if rd == 0 {
			err = e.generateDDLCreate()
		} else if rd < 20 {
			err = e.generateInsert()
		} else if rd < 40 {
			err = e.generateUpdate()
		} else {
			err = e.generateSelect()
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
	stmt, err := e.ss1.CreateTableStmt()
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
	stmt, err := e.ss1.SelectStmt(4)
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
	stmt, err := e.ss1.UpdateStmt()
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
	stmt, err := e.ss1.InsertStmtAST()
	if err != nil {
		return errors.Trace(err)
	}
	e.ch <- &types.SQL{
		SQLType: types.SQLTypeDMLInsert,
		SQLStmt: stmt,
	}
	return nil	
}
