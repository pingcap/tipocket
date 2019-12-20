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
	"github.com/pingcap/tipocket/pocket/config"
	"github.com/pingcap/tipocket/pocket/connection"
	"github.com/pingcap/tipocket/pocket/executor"
)

// Core is for random test scheduler
type Core struct {
	cfg        *config.Config
	executors  []*executor.Executor
	deadlockCh chan int
	// DSN here is for fetch schema only
	coreConn *connection.Connection
	coreExec *executor.Executor
}

// New creates a Core struct
func New(cfg *config.Config) *Core {
	return &Core{
		cfg: cfg,
	}
}

// Start test
func (c *Core) Start(ctx context.Context) error {
	if err := c.initCoreConnection(); err != nil {
		return errors.Trace(err)
	}
	if err := c.mustExec(); err != nil {
		return errors.Trace(err)
	}
	if err := c.initSubConnection(); err != nil {
		return errors.Trace(err)
	}
	if c.cfg.Options.Reproduce {
		c.reproduce()
	} else {
		go c.watchLock()
		go c.startCheckConsistency()
		c.generate(ctx)
	}
	return nil
}
