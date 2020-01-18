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
	"math/rand"
	"sync"
	"time"

	"github.com/pingcap/tipocket/pocket/executor"
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

const (
	maxExecuteTime = 4.0 // Second
)

func (c *Core) watchLock() {
	lastExecTime := time.Now()
	go func() {
		ticker := time.Tick(time.Second)
		for range ticker {
			if time.Now().Sub(lastExecTime).Seconds() > maxExecuteTime {
				// deadlock detected
				if c.ifLock {
					// if the core goroutine block in another lock, skip deadlock
					continue
				}
				c.Lock()
				c.resolveDeadLock(false)
				c.Unlock()
			}
		}
	}()
	for {
		c.order.Push(<-c.lockWatchCh)
		lastExecTime = time.Now()
	}
}

func (c *Core) resolveDeadLock(all bool) {
	// log.Info(e.order.GetHistroy())
	var wg sync.WaitGroup
	for c.order.Next() {
		for _, executor := range c.executors {
			if executor.GetID() == c.order.Val() {
				time.Sleep(10 * time.Millisecond)
				wg.Add(1)
				go c.resolveDeadLockOne(executor, &wg)
			}
		}
	}
	if all {
		wg.Add(1)
		go c.resolveDeadLockOne(c.nowExec, &wg)
	}
	c.order.Reset()
	wg.Wait()
}

func (c *Core) resolveDeadLockOne(executor *executor.Executor, wg *sync.WaitGroup) {
	if executor == nil {
		wg.Done()
		return
	}
	var sql types.SQL
	if rand.Float64() < 0.5 {
		sql = types.SQL{
			SQLType: types.SQLTypeTxnCommit,
			SQLStmt: "COMMIT",
		}
	} else {
		sql = types.SQL{
			SQLType: types.SQLTypeTxnRollback,
			SQLStmt: "ROLLBACK",
		}
	}
	// exec commit/rollback
	executor.ExecSQL(&sql)
	// wait for txn finish
	go func() {
		for true {
			time.Sleep(time.Millisecond)
			if !executor.IfTxn() {
				wg.Done()
				break
			}
		}
	}()
}
