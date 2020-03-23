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
	"math/rand"
	"sync"
	"time"

	"github.com/pingcap/tipocket/pkg/pocket/executor"
	"github.com/pingcap/tipocket/pkg/pocket/pkg/types"
)

const (
	maxExecuteTime   = 4 * time.Second
	maxFinishTxnTime = 4 * time.Second
)

func (c *Core) watchLock() {
	// check lock for too long only in Serialize mode
	lastExecTime := time.Now()
	// if not in Serialize mode, the lock should be resolved by other connections themselves
	// so only resolve lock for Serialize mode
	if c.cfg.Options.Serialize {
		go func() {
			ticker := time.Tick(time.Second)
			for range ticker {
				if time.Now().Sub(lastExecTime) > maxExecuteTime {
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
	}
	for {
		c.order.Push(<-c.lockWatchCh)
		lastExecTime = time.Now()
	}
}

func (c *Core) resolveDeadLock(all bool) {
	var wg sync.WaitGroup
	if all && c.nowExec != nil && c.order.Has(c.nowExec.GetID()) {
		wg.Add(1)
		c.resolveDeadLockOne(c.nowExec, &wg)
	}
	for c.order.Next() {
		for _, executor := range c.executors {
			if executor.GetID() == c.order.Val() {
				time.Sleep(50 * time.Millisecond)
				wg.Add(1)
				c.resolveDeadLockOne(executor, &wg)
			}
		}
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

	ch := make(chan struct{}, 1)
	ctx, cancel := context.WithTimeout(context.Background(), maxFinishTxnTime)
	defer cancel()

	go func() {
		// exec commit/rollback
		executor.ExecSQL(&sql)
		// wait for txn finish
		for true {
			time.Sleep(time.Millisecond)
			if !executor.IfTxn() {
				wg.Done()
				ch <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		// the transaction submit is locked, go to next session
		return
	case <-ch:
		return
	}
}
