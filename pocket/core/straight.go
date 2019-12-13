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

	executor.ExecSQL(sql)

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
