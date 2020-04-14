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
	"time"

	"github.com/ngaut/log"
)

// Select run select statement and return query result
func (c *Connection) Select(stmt string, args ...interface{}) ([][]*QueryItem, error) {
	start := time.Now()
	rows, err := c.db.Query(stmt, args...)
	if err != nil {
		c.logSQL(stmt, time.Now().Sub(start), err)
		return [][]*QueryItem{}, err
	}

	columnTypes, _ := rows.ColumnTypes()
	var result [][]*QueryItem

	for rows.Next() {
		var (
			rowResultSets []interface{}
			resultRow     []*QueryItem
		)
		for range columnTypes {
			rowResultSets = append(rowResultSets, new(interface{}))
		}
		if err := rows.Scan(rowResultSets...); err != nil {
			log.Info(err)
		}
		for index, resultItem := range rowResultSets {
			r := *resultItem.(*interface{})
			item := QueryItem{
				ValType: columnTypes[index],
			}
			if r != nil {
				bytes := r.([]byte)
				item.ValString = string(bytes)
			} else {
				item.Null = true
			}
			resultRow = append(resultRow, &item)
		}
		result = append(result, resultRow)
	}

	// TODO: make args and stmt together
	c.logSQL(stmt, time.Now().Sub(start), nil)
	return result, nil
}

// Update run update statement and return error
func (c *Connection) Update(stmt string) (int64, error) {
	var affectedRows int64
	start := time.Now()
	result, err := c.db.Exec(stmt)
	if err == nil {
		affectedRows, _ = result.RowsAffected()
	}
	c.logSQL(stmt, time.Now().Sub(start), err, affectedRows)
	return affectedRows, err
}

// Insert run insert statement and return error
func (c *Connection) Insert(stmt string) (int64, error) {
	var affectedRows int64
	start := time.Now()
	result, err := c.db.Exec(stmt)
	if err == nil {
		affectedRows, _ = result.RowsAffected()
	}
	c.logSQL(stmt, time.Now().Sub(start), err, affectedRows)
	return affectedRows, err
}

// Delete run delete statement and return error
func (c *Connection) Delete(stmt string) (int64, error) {
	var affectedRows int64
	start := time.Now()
	result, err := c.db.Exec(stmt)
	if err == nil {
		affectedRows, _ = result.RowsAffected()
	}
	c.logSQL(stmt, time.Now().Sub(start), err, affectedRows)
	return affectedRows, err
}

// ExecDDL do DDL actions
func (c *Connection) ExecDDL(query string, args ...interface{}) error {
	start := time.Now()
	_, err := c.db.Exec(query, args...)
	c.logSQL(query, time.Now().Sub(start), err)
	return err
}

// Exec do any exec actions
func (c *Connection) Exec(stmt string) error {
	_, err := c.db.Exec(stmt)
	return err
}

// Begin a txn
func (c *Connection) Begin() error {
	start := time.Now()
	err := c.db.Begin()
	c.logSQL("BEGIN", time.Now().Sub(start), err)
	return err
}

// Commit a txn
func (c *Connection) Commit() error {
	start := time.Now()
	err := c.db.Commit()
	c.logSQL("COMMIT", time.Now().Sub(start), err)
	return err
}

// Rollback a txn
func (c *Connection) Rollback() error {
	start := time.Now()
	err := c.db.Rollback()
	c.logSQL("ROLLBACK", time.Now().Sub(start), err)
	return err
}

// IfTxn show if in a transaction
func (c *Connection) IfTxn() bool {
	return c.db.IfTxn()
}

// GetBeginTime get the begin time of a transaction
// if not in transaction, return 0
func (c *Connection) GetBeginTime() time.Time {
	return c.db.GetBeginTime()
}

// GeneralLog turn on or turn off general_log in TiDB
func (c *Connection) GeneralLog(v int) error {
	_, err := c.db.Exec(fmt.Sprintf("set @@tidb_general_log=\"%d\"", v))
	return err
}

// SetTiFlashEngine sets the isolation to TiFlash in TiDB
func (c *Connection) SetTiFlashEngine() error {
	_, err := c.db.Exec(`SET @@session.tidb_isolation_read_engines = "tiflash"`)
	return err
}

// ShowDatabases list databases
func (c *Connection) ShowDatabases() ([]string, error) {
	res, err := c.Select("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	var dbs []string
	for _, db := range res {
		if len(db) == 1 {
			dbs = append(dbs, db[0].ValString)
		}
	}
	return dbs, nil
}
