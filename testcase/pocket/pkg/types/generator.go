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

// Generator interface to unify the usage of sqlsmith and sqlspider
type Generator interface {
	// ReloadSchema function read raw scheme
	// For each record
	// record[0] dbname
	// record[1] table name
	// record[2] table type
	// record[3] column name
	// record[4] column type
	ReloadSchema([][5]string) error
	// SetDB set operation database
	// the generated SQLs after this will be under this database
	SetDB(db string)
	// SetStable is a trigger for whether generate random or some database-basicinfo-dependent data
	// eg. SetStable(true) will disable both rand() and user() functions since they both make unstable result in different database
	SetStable(stable bool)
	// SetHint can control if hints would be generated or not
	SetHint(hint bool)
	// SelectStmt generate select SQL
	SelectStmt() string
	// InsertStmt generate insert SQL
	InsertStmt() string
	// UpdateStmt generate update SQL
	UpdateStmt() string
	// CreateTableStmt generate create table SQL
	CreateTableStmt() string
}
