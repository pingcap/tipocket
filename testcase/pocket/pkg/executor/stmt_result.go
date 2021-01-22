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
	"github.com/pingcap/tipocket/testcase/pocket/pkg/types"
)

type stmtResult struct {
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
