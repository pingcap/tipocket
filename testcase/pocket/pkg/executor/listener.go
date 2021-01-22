// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package executor

import (
	"github.com/juju/errors"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/types"
)

// ExecSQL add sql into exec queue
func (e *Executor) ExecSQL(sql *types.SQL) error {
	e.Lock()
	defer e.Unlock()
	// e.parseOnlineTable(sql)
	e.SQLCh <- sql
	return errors.Trace(<-e.ErrCh)
}

// func (e *Executor)parseOnlineTable(sql *types.SQL) {
// 	switch sql.SQLType {
// 	case types.SQLTypeTxnCommit, types.SQLTypeTxnRollback:
// 		e.OnlineTable = []string{}
// 	default:
// 		if sql.SQLTable != "" {
// 			e.OnlineTable = append(e.OnlineTable, sql.SQLTable)
// 		}
// 	}
// }
