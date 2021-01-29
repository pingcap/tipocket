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

package sqlsmith

import (
	"regexp"
	"strings"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/generator/generator"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/go-sqlsmith/types"
)

const typeRegex = `\(\d+\)`

// LoadSchema init schemas, tables and columns
// record[0] dbname
// record[1] table name
// record[2] table type
// record[3] column name
// record[4] column type
func (s *SQLSmith) LoadSchema(records [][5]string, indexes map[string][]string) {
	// init databases
	for _, record := range records {
		dbname := record[0]
		tableName := record[1]
		tableType := record[2]
		columnName := record[3]
		columnType := record[4]
		index, ok := indexes[tableName]
		if !ok {
			index = []string{}
		}
		if _, ok := s.Databases[dbname]; !ok {
			s.Databases[dbname] = &types.Database{
				Name:   dbname,
				Tables: make(map[string]*types.Table),
			}
		}
		if _, ok := s.Databases[dbname].Tables[tableName]; !ok {
			s.Databases[dbname].Tables[tableName] = &types.Table{
				DB:      dbname,
				Table:   tableName,
				Type:    tableType,
				Columns: make(map[string]*types.Column),
				Indexes: index,
			}
		}
		if _, ok := s.Databases[dbname].Tables[tableName].Columns[columnName]; !ok {
			s.Databases[dbname].Tables[tableName].Columns[columnName] = &types.Column{
				DB:     dbname,
				Table:  tableName,
				Column: columnName,
				// remove the data size in type definition
				DataType: regexp.MustCompile(typeRegex).ReplaceAllString(strings.ToLower(columnType), ""),
			}
		}
	}
}

// BeginWithOnlineTables begins a transaction with some online tables
func (s *SQLSmith) BeginWithOnlineTables(opt *generator.DMLOptions) []string {
	if !opt.OnlineTable {
		return []string{}
	}
	var (
		res    = []string{}
		db     = s.GetDB(s.GetCurrDBName())
		tables = db.RandTables()
	)
	for _, t := range tables {
		res = append(res, t.Table)
		t.Online = true
	}
	db.Online = true
	return res
}

// EndTransaction ends transaction and set every table offline
func (s *SQLSmith) EndTransaction() []string {
	var (
		res    = []string{}
		db     = s.GetDB(s.GetCurrDBName())
		tables = db.Tables
	)
	for _, t := range tables {
		res = append(res, t.Table)
		t.Online = false
	}
	db.Online = false
	return res
}

// setOnlineOtherTables is for avoiding online DDL
func (s *SQLSmith) setOnlineOtherTables(opt *generator.DDLOptions) {
	if opt.OnlineDDL {
		return
	}
	db := s.GetDB(s.currDB)
	for _, table := range opt.Tables {
		if db.Tables[table] != nil {
			db.Tables[table].OnlineOther = true
		}
	}
}

// freeOnlineOtherTables clear online tables obtain by other instances
func (s *SQLSmith) freeOnlineOtherTables() {
	db := s.GetDB(s.currDB)
	for _, table := range db.Tables {
		table.OnlineOther = false
	}
}
