package executor

import (
	"github.com/pingcap/tipocket/abclient/pkg/types"
)

// ExecSQL add sql into exec queue
func (e *Executor) ExecSQL(sql *types.SQL) {
	e.ch <- sql
	<-e.TxnReadyCh
}
