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

package connection

import (
	"github.com/juju/errors"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/logger"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/mysql"
)

// Option struct
type Option struct {
	Name          string
	Log           string
	Mute          bool
	GeneralLog    bool
	EnableTiFlash bool
}

// Connection define connection struct
type Connection struct {
	logger *logger.Logger
	db     *mysql.DBConnect
	opt    *Option
}

// New create Connection instance from dsn
func New(dsn string, opt *Option) (*Connection, error) {
	l, err := logger.New(opt.Name, opt.Log, opt.Mute)
	if err != nil {
		return nil, errors.Trace(err)
	}
	db, err := mysql.OpenDB(dsn, 1)
	if err != nil {
		return nil, errors.Trace(err)
	}
	c := &Connection{
		logger: l,
		db:     db,
		opt:    opt,
	}
	if err := c.Prepare(); err != nil {
		return nil, errors.Trace(err)
	}
	return c, nil
}

// Prepare connection
func (c *Connection) Prepare() error {
	if c.opt.GeneralLog {
		if err := errors.Trace(c.GeneralLog(1)); err != nil {
			return errors.Trace(err)
		}
	}
	if c.opt.EnableTiFlash {
		return errors.Trace(c.SetTiFlashEngine())
	}
	return nil
}

// CloseDB close connection
func (c *Connection) CloseDB() error {
	return c.db.CloseDB()
}

// ReConnect rebuild connection
func (c *Connection) ReConnect() error {
	if err := c.db.ReConnect(); err != nil {
		return err
	}
	return c.Prepare()
}
