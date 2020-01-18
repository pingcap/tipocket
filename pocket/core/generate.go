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
	"github.com/pingcap/tipocket/pocket/pkg/generator/generator"
	"github.com/pingcap/tipocket/pocket/pkg/types"
	"github.com/pingcap/tipocket/pocket/util"
)

func (c *Core) generate(ctx context.Context, readyCh *chan struct{}) error {
	if err := c.beforeGenerate(); err != nil {
		return errors.Trace(err)
	}

	log.Info("init done, start generate")
	*readyCh <- struct{}{}
	c.runGenerateSQL(ctx)
	return nil
}

func (c *Core) beforeGenerate() error {
	for i := 0; i < c.cfg.Options.InitTable; i++ {
		sql, e, err := c.generateDDLCreateTable()
		if err != nil {
			return errors.Trace(err)
		}
		if e != nil && sql != nil {
			c.execute(e, sql)
		}
		log.Infof("table %d generate", i)
	}
	for _, e := range c.executors {
		if err := e.ReloadSchema(); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (c *Core) runGenerateSQL(ctx context.Context) {
	if c.cfg.Options.Serialize {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				c.serializeGenerateSQL()
			}
		}
	} else {
		wg := sync.WaitGroup{}
		for _, e := range c.executors {
			wg.Add(1)
			go func(e *executor.Executor) {
				c.runGenerateSQLWithExecutor(ctx, e)
				wg.Done()
			}(e)
		}
		wg.Wait()
	}
}

func (c *Core) runGenerateSQLWithExecutor(ctx context.Context, e *executor.Executor) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			for c.ifLock {
				time.Sleep(time.Second)
			}
			c.generateSQLWithExecutor(e)
		}
	}
}

func (c *Core) generateSQLWithExecutor(e *executor.Executor) {
	var (
		sql *types.SQL
		err error
		rd  = util.Rd(350)
	)
	if rd == 0 {
		// sql, err = e.GenerateDDLCreateTable()
		sql, err = e.GenerateDDLAlterTable(c.getDDLOptions(e))
	} else if rd < 10 {
		sql, err = e.GenerateDDLAlterTable(c.getDDLOptions(e))
	} else if rd < 20 {
		sql, err = e.GenerateDDLCreateIndex(c.getDDLOptions(e))
	} else if rd < 40 {
		sql = &types.SQL{
			SQLType: types.SQLTypeTxnBegin,
			SQLStmt: "BEGIN",
		}
	} else if rd < 160 {
		sql, err = e.GenerateDMLUpdate()
	} else if rd < 180 {
		sql = &types.SQL{
			SQLType: types.SQLTypeTxnCommit,
			SQLStmt: "COMMIT",
		}
	} else if rd < 190 {
		sql = &types.SQL{
			SQLType: types.SQLTypeTxnRollback,
			SQLStmt: "ROLLBACK",
		}
	} else if rd < 220 {
		sql, err = e.GenerateDMLSelectForUpdate()
	} else if rd < 230 {
		sql, err = e.GenerateDMLDelete()
	} else if rd < 240 {
		sql = e.GenerateSleep()
	} else {
		// err = e.generateSelect()
		sql, err = e.GenerateDMLInsert()
	}

	if err != nil {
		// log.Fatalf("generate SQL error, %+v", errors.ErrorStack(err))
		log.Errorf("generate SQL error, %+v", errors.ErrorStack(err))
		return
	}
	if sql != nil {
		c.execute(e, sql)
	}
}

func (c *Core) serializeGenerateSQL() {
	c.Lock()
	c.Unlock()
	// TODO: SQL type rate config
	var (
		sql *types.SQL
		err error
		e   *executor.Executor
		rd  = util.Rd(350)
	)

	if rd == 0 {
		sql, e, err = c.generateDDLCreateTable()
	} else if rd < 10 {
		sql, e, err = c.generateDDLAlterTable()
	} else if rd < 20 {
		sql, e, err = c.generateDDLCreateIndex()
	} else if rd < 40 {
		sql, e, err = c.generateTxnBegin()
	} else if rd < 160 {
		sql, e, err = c.generateDMLUpdate()
	} else if rd < 180 {
		sql, e, err = c.generateTxnCommit()
	} else if rd < 190 {
		sql, e, err = c.generateTxnRollback()
	} else if rd < 220 {
		sql, e, err = c.generateDMLSelectForUpdate()
	} else if rd < 230 {
		sql, e, err = c.generateDMLDelete()
	} else if rd < 240 {
		sql, e, err = c.generateSleep()
	} else {
		// err = e.generateSelect()
		sql, e, err = c.generateDMLInsert()
	}

	if err != nil {
		// log.Fatalf("generate SQL error, %+v", errors.ErrorStack(err))
		log.Errorf("generate SQL error, %+v", errors.ErrorStack(err))
		return
	}
	c.nowExec = e
	if e != nil && sql != nil {
		c.execute(e, sql)
	}
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

func (c *Core) generateDDLAlterTable() (*types.SQL, *executor.Executor, error) {
	e := c.tryRandFreeExecutor()
	if e == nil {
		return nil, nil, nil
	}
	sql, err := e.GenerateDDLAlterTable(c.getDDLOptions(e))
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return sql, e, nil
}

func (c *Core) generateDDLCreateIndex() (*types.SQL, *executor.Executor, error) {
	e := c.tryRandFreeExecutor()
	if e == nil {
		return nil, nil, nil
	}
	sql, err := e.GenerateDDLCreateIndex(c.getDDLOptions(e))
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return sql, e, nil
}

func (c *Core) generateTxnBegin() (*types.SQL, *executor.Executor, error) {
	executor := c.randFreeExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	return &types.SQL{
		SQLType: types.SQLTypeTxnBegin,
		SQLStmt: "BEGIN",
	}, executor, nil
}

func (c *Core) generateTxnCommit() (*types.SQL, *executor.Executor, error) {
	executor := c.randBusyExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	return &types.SQL{
		SQLType: types.SQLTypeTxnCommit,
		SQLStmt: "COMMIT",
	}, executor, nil
}

func (c *Core) generateTxnRollback() (*types.SQL, *executor.Executor, error) {
	executor := c.randBusyExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	return &types.SQL{
		SQLType: types.SQLTypeTxnRollback,
		SQLStmt: "ROLLBACK",
	}, executor, nil
}

func (c *Core) generateDMLSelectForUpdate() (*types.SQL, *executor.Executor, error) {
	executor := c.tryRandBusyExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	sql, err := executor.GenerateDMLSelectForUpdate()
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return sql, executor, nil
}

func (c *Core) generateDMLInsert() (*types.SQL, *executor.Executor, error) {
	executor := c.tryRandBusyExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	sql, err := executor.GenerateDMLInsert()
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return sql, executor, nil
}

func (c *Core) generateDMLUpdate() (*types.SQL, *executor.Executor, error) {
	executor := c.tryRandBusyExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	sql, err := executor.GenerateDMLUpdate()
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return sql, executor, nil
}

func (c *Core) generateDMLDelete() (*types.SQL, *executor.Executor, error) {
	executor := c.tryRandBusyExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	sql, err := executor.GenerateDMLDelete()
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return sql, executor, nil
}

func (c *Core) generateSleep() (*types.SQL, *executor.Executor, error) {
	executor := c.tryRandBusyExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	return executor.GenerateSleep(), executor, nil
}

func (c *Core) randExecutor() *executor.Executor {
	return c.executors[util.Rd(len(c.executors))]
}

func (c *Core) randFreeExecutor() *executor.Executor {
	var notInTxns []*executor.Executor
	for _, e := range c.executors {
		if !e.IfTxn() {
			notInTxns = append(notInTxns, e)
		}
	}
	if len(notInTxns) == 0 {
		return nil
	}
	return notInTxns[util.Rd(len(notInTxns))]
}

func (c *Core) randBusyExecutor() *executor.Executor {
	var InTxns []*executor.Executor
	for _, e := range c.executors {
		if e.IfTxn() {
			InTxns = append(InTxns, e)
		}
	}
	if len(InTxns) == 0 {
		return nil
	}
	return InTxns[util.Rd(len(InTxns))]
}

// get free executor if exist, unless get a random executor
func (c *Core) tryRandFreeExecutor() *executor.Executor {
	if e := c.randFreeExecutor(); e != nil {
		return e
	}
	return c.randExecutor()
}

// get free executor if exist, unless get a random executor
func (c *Core) tryRandBusyExecutor() *executor.Executor {
	if e := c.randBusyExecutor(); e != nil {
		return e
	}
	return c.randExecutor()
}

func (c *Core) getDDLOptions(exec *executor.Executor) *generator.DDLOptions {
	tables := []string{}
	if c.cfg.Options.OnlineDDL {
		return &generator.DDLOptions{
			OnlineDDL: true,
			Tables:    tables,
		}
	}
	for _, e := range c.executors {
		if e == exec {
			continue
		}
		for _, o := range e.OnlineTable {
			for _, t := range tables {
				if o == t {
					continue
				}
			}
			tables = append(tables, o)
		}
	}
	return &generator.DDLOptions{
		OnlineDDL: false,
		Tables:    tables,
	}
}
