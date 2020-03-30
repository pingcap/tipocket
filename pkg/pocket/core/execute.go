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
	"time"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pkg/pocket/executor"
	"github.com/pingcap/tipocket/pkg/pocket/pkg/types"
)

const syncTimeout = 10 * time.Minute

func (c *Core) execute(e *executor.Executor, sql *types.SQL) {
	c.execMutex.Lock()
	defer c.execMutex.Unlock()
	// wait for execute finish
	// may not ignore the errors here
	if c.cfg.Options.Serialize && sql.ExecTime != 0 {
		c.Lock()
		go func() {
			time.Sleep(time.Duration(sql.ExecTime) * time.Second)
			c.Unlock()
		}()
	}
	_ = e.ExecSQL(sql)
	c.lockWatchCh <- e.GetID()
}

func (c *Core) getExecuteByID(id int) *executor.Executor {
	for _, e := range c.executors {
		if e.GetID() == id {
			return e
		}
	}
	return nil
}

func (c *Core) executeByID(id int, sql *types.SQL) {
	if e := c.getExecuteByID(id); e != nil {
		c.execute(e, sql)
	}
}

// coreInitDatabaseExecute is used only for create and drop database
// and the manipulated db should be c.dbname
func (c *Core) coreInitDatabaseExecute(sql *types.SQL) error {
	switch c.cfg.Mode {
	case "single":
		return errors.Trace(c.coreExec.GetConn().Exec(sql.SQLStmt))
	case "binlog", "tiflash":
		if err := c.coreExec.GetConn().Exec(sql.SQLStmt); err != nil {
			return errors.Trace(err)
		}
		return errors.Trace(c.waitSyncDatabase(sql.SQLType))
	case "abtest", "abtiflash":
		err1 := c.coreExec.GetConn1().Exec(sql.SQLStmt)
		err2 := c.coreExec.GetConn2().Exec(sql.SQLStmt)
		if err1 != nil {
			return errors.Trace(err1)
		}
		return errors.Trace(err2)
	default:
		panic("unhandled switch")
	}
}

func (c *Core) waitSyncDatabase(t types.SQLType) error {
	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return errors.Errorf("sync database timeout in %s", syncTimeout)
		case <-ticker.C:
			ifSync, err := c.ifSyncDatabase(t)
			if err != nil {
				return errors.Trace(err)
			}
			if ifSync {
				return nil
			}
		}
	}
}

func (c *Core) ifSyncDatabase(t types.SQLType) (bool, error) {
	dbs, err := c.coreExec.GetConn2().ShowDatabases()
	if err != nil {
		return false, errors.Trace(err)
	}

	hasDB := false
	for _, db := range dbs {
		if db == c.dbname {
			hasDB = true
		}
	}

	switch t {
	case types.SQLTypeCreateDatabase:
		return hasDB, nil
	case types.SQLTypeDropDatabase:
		return !hasDB, nil
	}

	panic("unreachable")
}
