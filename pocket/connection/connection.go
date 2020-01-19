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
	"fmt"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/pocket/pkg/logger"
	"github.com/pingcap/tipocket/pocket/pkg/mysql"
)

// Option struct
type Option struct {
	Log  string
	Mute bool
}

// Connection define connection struct
type Connection struct {
	logger *logger.Logger
	db     *mysql.DBConnect
}

// New create Connection instance from dsn
func New(dsn string, opt *Option) (*Connection, error) {
	l, err := logger.New(opt.Log, opt.Mute)
	if err != nil {
		return nil, errors.Trace(err)
	}
	db, err := mysql.OpenDB(dsn, 1)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &Connection{
		logger: l,
		db:     db,
	}, nil
}

// Prepare create test database
func (c *Connection) Prepare(db string) {
	c.db.MustExec(fmt.Sprintf(dropDatabaseSQL, db))
	c.db.MustExec(fmt.Sprintf(createDatabaseSQL, db))
}

// CloseDB close connection
func (c *Connection) CloseDB() error {
	return c.db.CloseDB()
}

// ReConnect rebuild connection
func (c *Connection) ReConnect() error {
	return c.db.ReConnect()
}
