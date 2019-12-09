package executor

import (
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

type stmtResult struct{
	sql *types.SQL
	err error
}

func (e *Executor) prewriteLog(sql *types.SQL) {
	e.Lock()
	defer e.Unlock()
	e.stmts = append(e.stmts, sql)
}

func (e *Executor) commitLog(sql *types.SQL, err error) {
	e.Lock()
	defer e.Unlock()
	// if len(e.stmts) == 0 {
	// 	panic("write log before prewrite")
	// }

	// if stmt == e.stmts[0] {
	// 	e.stmts = e.stmts[1:]
	// }


	// if err != nil {
	// 	e.logger.Infof("[FAIL] Exec SQL %s error %v", stmt, err)
	// } else {
	// 	e.logger.Infof("[SUCCESS] Exec SQL %s success", stmt)
	// }
}
