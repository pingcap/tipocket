package connection

import (
	"fmt"
	"regexp"
	"github.com/juju/errors"
)

var binlogSyncTablePattern = regexp.MustCompile(`^t[0-9]+$`)

// FetchTables table list
func (c *Connection) FetchTables(db string) ([]string, error) {
	tableSlice := []string{}

	tables, err := c.db.Query(schemaSQL)
	if err != nil {
		return []string{}, errors.Trace(err)
	}

	// fetch tables need to be described
	for tables.Next() {
		var schemaName, tableName string
		if err = tables.Scan(&schemaName, &tableName, new(interface{})); err != nil {
			return []string{}, errors.Trace(err)
		}
		if schemaName == db {
			tableSlice = append(tableSlice, tableName)
		}
	}
	return tableSlice, nil
}

// FetchSchema get schema of given database from database
func (c *Connection) FetchSchema(db string) ([][5]string, error) {
	var (
		schema [][5]string
		tablesInDB [][3]string
	)
	tables, err := c.db.Query(schemaSQL)
	if err != nil {
		return schema, errors.Trace(err)
	}

	// fetch tables need to be described
	for tables.Next() {
		var schemaName, tableName, tableType string
		if err = tables.Scan(&schemaName, &tableName, &tableType); err != nil {
			return [][5]string{}, errors.Trace(err)
		}
		// do not generate SQL about sync table
		// because this can not be reproduced from logs
		if schemaName == db && !ifBinlogSyncTable(tableName) {
			tablesInDB = append(tablesInDB, [3]string{schemaName, tableName, tableType})
		}
	}

	// desc tables
	for _, table := range tablesInDB {
		var (
			schemaName = table[0]
			tableName = table[1]
			tableType = table[2]
		)
		columns, err := c.FetchColumns(schemaName, tableName)
		if err != nil {
			return [][5]string{}, errors.Trace(err)
		}
		for _, column := range columns {
			schema = append(schema, [5]string{schemaName, tableName, tableType, column[0], column[1]})
		}
	}
	return schema, nil
}

// FetchColumns get columns for given table
func (c *Connection) FetchColumns(db, table string) ([][2]string, error) {
	var columns [][2]string
	res, err := c.db.Query(fmt.Sprintf(tableSQL, db, table))
	if err != nil {
		return [][2]string{}, errors.Trace(err)
	}
	for res.Next() {
		var columnName, columnType string
		var col1, col2, col3, col4 interface{}
		if err = res.Scan(&columnName, &columnType, &col1, &col2, &col3, &col4); err != nil {
			return [][2]string{}, errors.Trace(err)
		}
		columns = append(columns, [2]string{columnName, columnType})
	}
	return columns, nil
}

// FetchIndexes get indexes for given table
func (c *Connection) FetchIndexes(db, table string) ([]string, error) {
	var indexes []string
	res, err := c.db.Query(fmt.Sprintf(indexSQL, db, table))
	if err != nil {
		return []string{}, errors.Trace(err)
	}
	for res.Next() {
		var keyname string
		var col1, col2, col4, col5, col6, col7, col8, col9, col10, col11, col12, col13 interface{}
		if err = res.Scan(&col1, &col2, &keyname, &col4, &col5, &col6, &col7, &col8, &col9, &col10, &col11, &col12, &col13); err != nil {
			return []string{}, errors.Trace(err)
		}
		indexes = append(indexes, keyname)
	}
	return indexes, nil
}

func ifBinlogSyncTable(t string) bool {
	return binlogSyncTablePattern.MatchString(t)
}
