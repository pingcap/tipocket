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
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/executor"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/generator/generator"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/types"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/util"
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
		log.Infof("table %s generate", sql.SQLTable)
	}

	if c.coreExec.TiFlash {
		tables, err := c.coreConn.FetchTables("pocket")
		if err != nil {
			return errors.Trace(err)
		}
		for _, t := range tables {
			if err := c.coreExec.WaitTiFlashTableSync(t); err != nil {
				log.Infof("table %s sync to tiflash failed, error is %v", t, err)
				return errors.Trace(err)
			}
			log.Infof("table %s sync to tiflash completed", t)
		}
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

func (c *Core) randSQLType() types.SQLType {
	// FIXME: use reflect and calc it only once in init
	all := 0
	all = all + c.cfg.Generator.SQLSmith.TxnBegin
	all = all + c.cfg.Generator.SQLSmith.TxnCommit
	all = all + c.cfg.Generator.SQLSmith.TxnRollback
	all = all + c.cfg.Generator.SQLSmith.DDLCreateTable
	all = all + c.cfg.Generator.SQLSmith.DDLAlterTable
	all = all + c.cfg.Generator.SQLSmith.DDLCreateIndex
	all = all + c.cfg.Generator.SQLSmith.DMLSelect
	all = all + c.cfg.Generator.SQLSmith.DMLSelectForUpdate
	all = all + c.cfg.Generator.SQLSmith.DMLDelete
	all = all + c.cfg.Generator.SQLSmith.DMLUpdate
	all = all + c.cfg.Generator.SQLSmith.DMLInsert
	all = all + c.cfg.Generator.SQLSmith.Sleep

	val := util.Rd(all)
	if val < c.cfg.Generator.SQLSmith.TxnBegin {
		return types.SQLTypeTxnBegin
	}
	val = val - c.cfg.Generator.SQLSmith.TxnBegin

	if val < c.cfg.Generator.SQLSmith.TxnCommit {
		return types.SQLTypeTxnCommit
	}
	val = val - c.cfg.Generator.SQLSmith.TxnCommit

	if val < c.cfg.Generator.SQLSmith.TxnRollback {
		return types.SQLTypeTxnRollback
	}
	val = val - c.cfg.Generator.SQLSmith.TxnRollback

	if val < c.cfg.Generator.SQLSmith.DDLCreateTable {
		return types.SQLTypeDDLCreateTable
	}
	val = val - c.cfg.Generator.SQLSmith.DDLCreateTable

	if val < c.cfg.Generator.SQLSmith.DDLAlterTable {
		return types.SQLTypeDDLAlterTable
	}
	val = val - c.cfg.Generator.SQLSmith.DDLAlterTable

	if val < c.cfg.Generator.SQLSmith.DDLCreateIndex {
		return types.SQLTypeDDLCreateIndex
	}
	val = val - c.cfg.Generator.SQLSmith.DDLCreateIndex

	if val < c.cfg.Generator.SQLSmith.DMLSelect {
		return types.SQLTypeDMLSelect
	}
	val = val - c.cfg.Generator.SQLSmith.DMLSelect

	if val < c.cfg.Generator.SQLSmith.DMLSelectForUpdate {
		return types.SQLTypeDMLSelectForUpdate
	}
	val = val - c.cfg.Generator.SQLSmith.DMLSelectForUpdate

	if val < c.cfg.Generator.SQLSmith.DMLDelete {
		return types.SQLTypeDMLDelete
	}
	val = val - c.cfg.Generator.SQLSmith.DMLDelete

	if val < c.cfg.Generator.SQLSmith.DMLUpdate {
		return types.SQLTypeDMLUpdate
	}
	val = val - c.cfg.Generator.SQLSmith.DMLUpdate

	if val < c.cfg.Generator.SQLSmith.DMLDelete {
		return types.SQLTypeDMLDelete
	}
	val = val - c.cfg.Generator.SQLSmith.DMLDelete

	if val < c.cfg.Generator.SQLSmith.DMLInsert {
		return types.SQLTypeDMLInsert
	}
	val = val - c.cfg.Generator.SQLSmith.DMLInsert

	if val < c.cfg.Generator.SQLSmith.Sleep {
		return types.SQLTypeSleep
	}
	val = val - c.cfg.Generator.SQLSmith.Sleep

	return types.SQLTypeUnknown
}

func (c *Core) generateSQLWithExecutor(e *executor.Executor) {
	var (
		sql *types.SQL
		err error
	)

	switch c.randSQLType() {
	case types.SQLTypeDDLCreateTable:
		sql, err = e.GenerateDDLCreateTable()
	case types.SQLTypeDDLAlterTable:
		sql, err = e.GenerateDDLAlterTable(c.getDDLOptions(e))
	case types.SQLTypeDDLCreateIndex:
		sql, err = e.GenerateDDLCreateIndex(c.getDDLOptions(e))
	case types.SQLTypeTxnBegin:
		sql = e.GenerateTxnBegin()
	case types.SQLTypeTxnCommit:
		sql = e.GenerateTxnCommit()
	case types.SQLTypeTxnRollback:
		sql = e.GenerateTxnRollback()
	case types.SQLTypeDMLInsert:
		sql, err = e.GenerateDMLInsert()
	case types.SQLTypeDMLUpdate:
		sql, err = e.GenerateDMLUpdate()
	case types.SQLTypeDMLSelect:
		sql, err = e.GenerateDMLSelect()
	case types.SQLTypeDMLSelectForUpdate:
		sql, err = e.GenerateDMLSelectForUpdate()
	case types.SQLTypeDMLDelete:
		sql, err = e.GenerateDMLDelete()
	case types.SQLTypeSleep:
		sql = e.GenerateSleep()
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
	)

	switch c.randSQLType() {
	case types.SQLTypeDDLCreateTable:
		sql, e, err = c.generateDDLCreateTable()
	case types.SQLTypeDDLAlterTable:
		sql, e, err = c.generateDDLAlterTable()
	case types.SQLTypeDDLCreateIndex:
		sql, e, err = c.generateDDLCreateIndex()
	case types.SQLTypeTxnBegin:
		sql, e, err = c.generateTxnBegin()
	case types.SQLTypeTxnCommit:
		sql, e, err = c.generateTxnCommit()
	case types.SQLTypeTxnRollback:
		sql, e, err = c.generateTxnRollback()
	case types.SQLTypeDMLInsert:
		sql, e, err = c.generateDMLInsert()
	case types.SQLTypeDMLUpdate:
		sql, e, err = c.generateDMLUpdate()
	case types.SQLTypeDMLSelect:
		sql, e, err = c.generateDMLSelect()
	case types.SQLTypeDMLSelectForUpdate:
		sql, e, err = c.generateDMLSelectForUpdate()
	case types.SQLTypeDMLDelete:
		sql, e, err = c.generateDMLDelete()
	case types.SQLTypeSleep:
		sql, e, err = c.generateSleep()
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
	return executor.GenerateTxnBegin(), executor, nil
}

func (c *Core) generateTxnCommit() (*types.SQL, *executor.Executor, error) {
	executor := c.randBusyExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	return executor.GenerateTxnCommit(), executor, nil
}

func (c *Core) generateTxnRollback() (*types.SQL, *executor.Executor, error) {
	executor := c.randBusyExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	return executor.GenerateTxnRollback(), executor, nil
}

func (c *Core) generateDMLSelect() (*types.SQL, *executor.Executor, error) {
	executor := c.tryRandBusyExecutor()
	if executor == nil {
		return nil, nil, nil
	}
	sql, err := executor.GenerateDMLSelect()
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return sql, executor, nil
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
