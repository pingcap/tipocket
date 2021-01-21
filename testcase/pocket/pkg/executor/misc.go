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
	"fmt"

	"github.com/juju/errors"
)

// PrintSchema print schema information and return
func (e *Executor) PrintSchema() error {
	schema, err := e.conn1.FetchSchema(e.dbname)
	if err != nil {
		return errors.Trace(err)
	}
	for _, item := range schema {
		fmt.Printf("{\"%s\", \"%s\", \"%s\", \"%s\", \"%s\"},\n", item[0], item[1], item[2], item[3], item[4])
	}
	return nil
}

func (e *Executor) logStmtTodo(stmt string) {
	e.logger.Infof("[TODO] Exec SQL %s", stmt)
}

func (e *Executor) logStmtResult(stmt string, err error) {
	if err != nil {
		e.logger.Infof("[FAIL] Exec SQL %s error %v", stmt, err)
	} else {
		e.logger.Infof("[SUCCESS] Exec SQL %s success", stmt)
	}
}
