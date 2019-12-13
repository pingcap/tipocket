package executor

import (
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

// ExecSQL add sql into exec queue
func (e *Executor) ExecSQL(sql *types.SQL) {
	<-e.TxnReadyCh
	e.ch <- sql
}
