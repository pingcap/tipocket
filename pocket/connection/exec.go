package connection

import (
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
func (c *Connection) Update(stmt string) error {
	start := time.Now()
	_, err := c.db.Exec(stmt)
	c.logSQL(stmt, time.Now().Sub(start), err)
	return err
}

// Insert run insert statement and return error
func (c *Connection) Insert(stmt string) error {
	start := time.Now()
	_, err := c.db.Exec(stmt)
	c.logSQL(stmt, time.Now().Sub(start), err)
	return err
}

// Delete run delete statement and return error
func (c *Connection) Delete(stmt string) error {
	start := time.Now()
	_, err := c.db.Exec(stmt)
	c.logSQL(stmt, time.Now().Sub(start), err)
	return err
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
