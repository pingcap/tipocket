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
	"github.com/pingcap/tipocket/pocket/executor"
	"github.com/pingcap/tipocket/pocket/pkg/types"
)

func (c *Core) execute(e *executor.Executor, sql *types.SQL) {
}

func (c *Core) randExecutor() *executor.Executor {
	return c.executors[rand.Intn(len(c.executors))]
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
	return notInTxns[rand.Intn(len(notInTxns))]
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
	return InTxns[rand.Intn(len(InTxns))]
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
