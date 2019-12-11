package core

import (
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pocket/executor"
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

// straight is for execute SQLs and transactions directly

// ExecStraight exec SQL in given executor
func (e *Executor) ExecStraight(sql *types.SQL, node int) {
	var (
		executor = e.findExecutor(node)
		err error
	)

	if executor == nil {
		log.Fatalf("can not find executor, id %d", node)
	}
	// log.Infof("Node %d, Type %s, Exec %s", node, sql.SQLType, sql.SQLStmt)
	e.logger.Infof("Node %d, Type %s, Exec %s", node, sql.SQLType, sql.SQLStmt)

	switch sql.SQLType {
	case types.SQLTypeDMLSelect:
		err = executor.Select(sql.SQLStmt)
	case types.SQLTypeDMLInsert:
		err = executor.Insert(sql.SQLStmt)
	case types.SQLTypeDMLUpdate:
		err = executor.Update(sql.SQLStmt)
	case types.SQLTypeDMLDelete:
		err = executor.Delete(sql.SQLStmt)
	case types.SQLTypeDDLCreate:
		err = executor.CreateTable(sql.SQLStmt)
	case types.SQLTypeDDLAlterTable:
		err = executor.AlterTable(sql.SQLStmt)
	case types.SQLTypeDDLCreateIndex:
		err = executor.CreateIndex(sql.SQLStmt)
	case types.SQLTypeTxnBegin:
		err = executor.TxnBegin()
	case types.SQLTypeTxnCommit:
		err = executor.TxnCommit()
	case types.SQLTypeTxnRollback:
		err = executor.TxnRollback()
	default:
		panic("unhandled exec straight")
	}

	switch sql.SQLType {
	case types.SQLTypeDDLCreate:
		executor.TxnCommit()
	}

	if err != nil {
		log.Infof("error, %+v\n", err)
	} else {
		log.Infof("success\n")
	}
}

func (e *Executor) findExecutor(node int) *executor.Executor {
	for _, e := range e.executors {
		if e.GetID() == node {
			return e
		}
	}
	return nil
}
