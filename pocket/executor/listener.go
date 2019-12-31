package executor

import (
	"github.com/juju/errors"
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

// ExecSQL add sql into exec queue
func (e *Executor) ExecSQL(sql *types.SQL) error {
	e.Lock()
	defer e.Unlock()
	e.SQLCh <- sql
	return errors.Trace(<-e.ErrCh)
}
