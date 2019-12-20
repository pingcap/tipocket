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

package core

import (
	"context"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pocket/executor"
	"github.com/pingcap/tipocket/pocket/pkg/types"
	"github.com/pingcap/tipocket/pocket/util"
)

func (c *Core) generate(ctx context.Context) {
	for{
	select {
	case <-ctx.Done():
		return
	default:
		c.generateSQL()
	}
	}
}

func (c *Core) generateSQL() {
	// TODO: SQL type rate config
	var (
		sql  *types.SQL
		err  error
		e    *executor.Executor
		rd = util.Rd(300)
	)

	if rd == 0 {
		sql, e, err = c.generateDDLCreateTable()
	}

	if err != nil {
		log.Fatalf("generate SQL error, %+v", errors.ErrorStack(err))
	}

	c.execute(e, sql)
}

func (c *Core) generateDDLCreateTable() (*types.SQL, *executor.Executor, error) {
	executor := c.tryRandFreeExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	sql, err := executor.GenerateDDLCreateTable()
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return sql, executor, nil
}
