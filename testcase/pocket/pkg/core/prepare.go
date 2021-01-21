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
	"regexp"
	"time"

	"github.com/juju/errors"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/types"
)

var (
	dsnPattern   = regexp.MustCompile(`([a-z0-9]+):@tcp\(([0-9.]+):([0-9]+)\)/([0-9a-zA-Z_]+)`)
	mustExecSQLs = []string{
		`SET @@GLOBAL.SQL_MODE="NO_ENGINE_SUBSTITUTION"`,
		// `SET @@GLOBAL.TIME_ZONE = "+8:00"`,
	}
	mustExecSQLsIgnoreErr = []string{
		`SET @@GLOBAL.TIDB_TXN_MODE="pessimistic"`,
		`SET @@GLOBAL.explicit_defaults_for_timestamp=1`,
	}
)

func removeDSNSchema(dsn string) string {
	m := dsnPattern.FindStringSubmatch(dsn)
	if len(m) != 5 {
		return dsn
	}
	return fmt.Sprintf("%s:@tcp(%s:%s)/", m[1], m[2], m[3])
}

func (c *Core) prepare() error {
	if err := c.parseDSN(); err != nil {
		return errors.Trace(err)
	}
	if err := c.mustExec(); err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(c.prepareSchema())
}

func (c *Core) parseDSN() error {
	m := dsnPattern.FindStringSubmatch(c.cfg.DSN1)
	if len(m) != 5 {
		return errors.Errorf("invalid dsn %s", c.cfg.DSN1)
	}
	c.dbname = m[4]
	return nil
}

func (c *Core) mustExec() error {
	for _, sql := range mustExecSQLsIgnoreErr {
		c.coreExec.ExecIgnoreErr(sql)
	}
	for _, sql := range mustExecSQLs {
		if err := c.coreExec.Exec(sql); err != nil {
			return errors.Trace(err)
		}
	}

	// sleep 10s waiting for global variable becoming effective
	time.Sleep(10 * time.Second)
	return nil
}

func (c *Core) prepareSchema() error {
	dbs, err := c.coreConn.FetchDatabases()
	if err != nil {
		return errors.Trace(err)
	}
	needClearSchema := false
	for _, db := range dbs {
		if db == c.dbname {
			needClearSchema = true
		}
	}
	if needClearSchema {
		return errors.Trace(c.clearSchema())
	}
	return errors.Trace(c.createSchema())
}

func (c *Core) clearSchema() error {
	if !c.cfg.Options.ClearDB {
		return nil
	}
	if err := c.coreInitDatabaseExecute(&types.SQL{
		SQLStmt: fmt.Sprintf("DROP DATABASE %s", c.dbname),
		SQLType: types.SQLTypeDropDatabase,
	}); err != nil {
		return errors.Trace(err)
	}
	return errors.Trace(c.createSchema())
}

func (c *Core) createSchema() error {
	return errors.Trace(c.coreInitDatabaseExecute(&types.SQL{
		SQLStmt: fmt.Sprintf("CREATE DATABASE %s", c.dbname),
		SQLType: types.SQLTypeCreateDatabase,
	}))
}
