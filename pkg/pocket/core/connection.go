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

	"github.com/pingcap/tipocket/pkg/pocket/executor"
)

func (c *Core) generateExecutorOption(id int) *executor.Option {
	var suffix string
	if id > 0 {
		suffix = fmt.Sprintf("-%d", id)
	}
	opt := executor.Option{
		ID:         id,
		Log:        c.cfg.Options.Path,
		LogSuffix:  suffix,
		Stable:     c.cfg.Options.Stable,
		Mute:       c.cfg.Options.Reproduce,
		OnlineDDL:  c.cfg.Options.OnlineDDL,
		GeneralLog: c.cfg.Options.GeneralLog,
		Hint:       c.cfg.Options.EnableHint,
	}
	return &opt
}

func (c *Core) initConnectionWithoutSchema(id int) (*executor.Executor, error) {
	var (
		e       *executor.Executor
		err     error
		tiFlash bool
	)

	if c.cfg.Mode == "abtiflash" || c.cfg.Mode == "tiflash" {
		tiFlash = true
	}

	switch c.cfg.Mode {
	case "single", "tiflash":
		e, err = executor.New(removeDSNSchema(c.cfg.DSN1), c.generateExecutorOption(id), tiFlash)
		if err != nil {
			return nil, errors.Trace(err)
		}
	case "abtest", "binlog", "abtiflash":
		e, err = executor.NewABTest(removeDSNSchema(c.cfg.DSN1),
			removeDSNSchema(c.cfg.DSN2),
			c.generateExecutorOption(id), tiFlash)
		if err != nil {
			return nil, errors.Trace(err)
		}
	default:
		return nil, errors.Errorf("unhandled mode, %s", c.cfg.Mode)
	}
	return e, nil
}

func (c *Core) initConnection(id int) (*executor.Executor, error) {
	var (
		e       *executor.Executor
		err     error
		mode    string
		tiFlash bool
	)

	switch c.cfg.Mode {
	case "single", "tiflash":
		mode = "single"
	case "abtest", "abtiflash":
		mode = "abtest"
	case "binlog":
		if id == 0 {
			mode = "abtest"
		} else {
			mode = "single"
		}
	}

	if c.cfg.Mode == "abtiflash" || c.cfg.Mode == "tiflash" {
		tiFlash = true
	}

	if mode == "single" {
		e, err = executor.New(c.cfg.DSN1, c.generateExecutorOption(id), tiFlash)
		if err != nil {
			return nil, errors.Trace(err)
		}
	} else if mode == "abtest" {
		e, err = executor.NewABTest(c.cfg.DSN1, c.cfg.DSN2, c.generateExecutorOption(id), tiFlash)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	return e, nil
}

func (c *Core) initCoreConnectionWithoutSchema() error {
	e, err := c.initConnectionWithoutSchema(0)
	if err != nil {
		return errors.Trace(err)
	}
	c.coreExec = e
	c.coreConn = e.GetConn()
	return nil
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

func (c *Core) initCompareConnection() (*executor.Executor, error) {
	var tiFlash bool
	if c.cfg.Mode == "abtiflash" || c.cfg.Mode == "tiflash" {
		tiFlash = true
	}
	opt := c.generateExecutorOption(0)
	opt.Mute = true
	return executor.NewABTest(c.cfg.DSN1, c.cfg.DSN2, opt, tiFlash)
}
