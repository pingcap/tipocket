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
	"fmt"
	"github.com/juju/errors"
	"github.com/pingcap/tipocket/pocket/executor"
)

func (c *Core) generateExecutorOption(id int) *executor.Option {
	var suffix string
	if id > 0 {
		suffix = fmt.Sprintf("-%d", id)
	}
	opt := executor.Option{
		ID: id,
		Log: c.cfg.Options.Path,
		LogSuffix: suffix,
		Stable: c.cfg.Options.Stable,
		Mute: !c.cfg.Options.Reproduce,
	}
	return &opt
}

func (c *Core) initConnection(id int) (*executor.Executor, error) {
	var (
		e *executor.Executor
		err error
	)
	switch c.cfg.Mode {
	case "single":
		e, err = executor.New(c.cfg.Dsn1, c.generateExecutorOption(id))
		if err != nil {
			return nil, errors.Trace(err)
		}
	case "abtest", "binlog":
		e, err = executor.NewABTest(c.cfg.Dsn1, c.cfg.Dsn2, c.generateExecutorOption(id))
		if err != nil {
			return nil, errors.Trace(err)
		}
	default:
		return nil, errors.Errorf("unhandled mode, %s", c.cfg.Mode)
	}
	return e, nil
}

func (c *Core) initCoreConnection() error {
	e, err := c.initConnection(0)
	if err != nil {
		return errors.Trace(err)
	}
	c.coreExec = e
	c.coreConn = e.GetConn()
	return nil
}

func (c *Core) initSubConnection() error {
	concurrency := c.cfg.Options.Concurrency
	if concurrency <= 0 {
		return errors.Errorf("invalid concurrency, is or less than 0, got %d", concurrency)
	}
	for i := 0; i < concurrency; i++ {
		e, err := c.initConnection(i + 1)
		if err != nil {
			return errors.Trace(err)
		}
		c.executors = append(c.executors, e)
	}
	return nil
}
