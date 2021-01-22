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

package types

import (
	"github.com/pingcap/tipocket/testcase/pocket/pkg/go-sqlsmith/util"
)

// Database defines database database
type Database struct {
	Name   string
	Online bool
	Tables map[string]*Table
}

// RandTables rand tables
func (d *Database) RandTables() []*Table {
	tables := []*Table{}
	for _, t := range d.Tables {
		if util.RdFloat64() < 0.3 {
			tables = append(tables, t)
		}
	}
	if len(tables) == 0 {
		for _, t := range d.Tables {
			tables = append(tables, t)
			return tables
		}
	}
	return tables
}
