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

// SQLType enums for SQL types
type SQLType int

// SQLTypeDMLSelect
const (
	SQLTypeUnknown SQLType = iota
	SQLTypeReloadSchema
	SQLTypeDMLSelect
	SQLTypeDMLUpdate
	SQLTypeDMLInsert
	SQLTypeDMLDelete
	SQLTypeDDLCreateTable
	SQLTypeDDLAlterTable
	SQLTypeDDLCreateIndex
	SQLTypeTxnBegin
	SQLTypeTxnCommit
	SQLTypeTxnRollback
	SQLTypeExec
	SQLTypeExit
)

// SQL struct
type SQL struct {
	SQLType     SQLType
	SQLStmt     string
	SQLTable    string
}

func (t SQLType) String() string {
	switch t {
	case SQLTypeReloadSchema:
		return "SQLTypeReloadSchema"
	case SQLTypeDMLSelect:
		return "SQLTypeDMLSelect"
	case SQLTypeDMLUpdate:
		return "SQLTypeDMLUpdate"
	case SQLTypeDMLInsert:
		return "SQLTypeDMLInsert"
	case SQLTypeDMLDelete:
		return "SQLTypeDMLDelete"
	case SQLTypeDDLCreateTable:
		return "SQLTypeDDLCreateTable"
	case SQLTypeDDLAlterTable:
		return "SQLTypeDDLAlterTable"
	case SQLTypeDDLCreateIndex:
		return "SQLTypeDDLCreateIndex"
	case SQLTypeTxnBegin:
		return "SQLTypeTxnBegin"
	case SQLTypeTxnCommit:
		return "SQLTypeTxnCommit"
	case SQLTypeTxnRollback:
		return "SQLTypeTxnRollback"
	case SQLTypeExec:
		return "SQLTypeExec"
	case SQLTypeExit:
		return "SQLTypeExit"
	default:
		return "SQLTypeUnknown"
	}
}
