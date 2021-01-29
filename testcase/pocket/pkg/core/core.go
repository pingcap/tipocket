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

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/config"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/connection"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/executor"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/types"
)

// Core is for random test scheduler
type Core struct {
	// global config
	cfg *config.Config
	// DSN here is for fetch schema only
	coreConn *connection.Connection
	coreExec *executor.Executor
	// core properties
	dbname      string
	nowExec     *executor.Executor
	executors   []*executor.Executor
	lockWatchCh chan int
	order       *types.Order
	// lock
	mutex     sync.Mutex
	execMutex sync.Mutex
	ifLock    bool
}

// New creates a Core struct
func New(cfg *config.Config) *Core {
	return &Core{
		cfg:         cfg,
		lockWatchCh: make(chan int),
		order:       types.NewOrder(),
	}
}

// Start test
func (c *Core) Start(ctx context.Context) error {
	if err := c.initCoreConnectionWithoutSchema(); err != nil {
		return errors.Trace(err)
	}
	if err := c.prepare(); err != nil {
		return errors.Trace(err)
	}
	if err := c.initCoreConnection(); err != nil {
		return errors.Trace(err)
	}
	if err := c.initSubConnection(); err != nil {
		return errors.Trace(err)
	}
	go c.watchLock()
	if c.cfg.Options.Reproduce {
		return c.reproduce(ctx)
	}
	initTableReadyCh := make(chan struct{}, 1)
	go func() {
		<-initTableReadyCh
		go c.startCheckConsistency(ctx)
	}()
	return errors.Trace(c.generate(ctx, &initTableReadyCh))
}
