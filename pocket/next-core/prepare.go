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
	"github.com/juju/errors"
)

var (
	mustExecSQLs = []string{
		`SET @@GLOBAL.SQL_MODE="NO_ENGINE_SUBSTITUTION"`,
		`SET @@GLOBAL.TIME_ZONE = "+8:00"`,
	}
	mustExecSQLsIgnoreErr = []string{
		`SET @@GLOBAL.TIDB_TXN_MODE="pessimistic"`,
		`SET @@GLOBAL.explicit_defaults_for_timestamp=1`,
	}
)

func (c *Core) mustExec() error {
	for _, sql := range mustExecSQLsIgnoreErr {
		c.coreExec.ExecIgnoreErr(sql)
	}
	for _, sql := range mustExecSQLs {
		if err := c.coreExec.Exec(sql); err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (c *Core) clearSchema() error {
	if !c.cfg.Options.ClearDB {
		return nil
	}
	return errors.NotImplementedf("not implemented yet")
}
