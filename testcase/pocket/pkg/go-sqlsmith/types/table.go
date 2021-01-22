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
	"sort"
	"strings"

	"github.com/pingcap/tipocket/testcase/pocket/pkg/go-sqlsmith/util"
)

// Table defines database table
type Table struct {
	DB          string
	Table       string
	OriginTable string
	Type        string
	Columns     map[string]*Column
	Indexes     []string
	// Online is for self obtain,
	// which means this table is online and will be manipulated in a txn
	Online bool
	// OnlineOther is for other instances obtain,
	// which means this table is being manipulated in other txns and should not be a DDL table
	OnlineOther    bool
	InnerTableList []*Table
}

type byColumn []*Column

func (a byColumn) Len() int      { return len(a) }
func (a byColumn) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byColumn) Less(i, j int) bool {
	var (
		bi = []byte(a[i].Column)
		bj = []byte(a[j].Column)
	)

	for i := 0; i < min(len(bi), len(bj)); i++ {
		if bi[i] != bj[i] {
			return bi[i] < bj[i]
		}
	}
	return len(bi) < len(bj)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Clone copy table struct
func (t *Table) Clone() *Table {
	newTable := Table{
		DB:             t.DB,
		Table:          t.Table,
		OriginTable:    t.OriginTable,
		Type:           t.Type,
		Columns:        make(map[string]*Column),
		Indexes:        t.Indexes,
		Online:         t.Online,
		OnlineOther:    t.OnlineOther,
		InnerTableList: t.InnerTableList,
	}
	for k, column := range t.Columns {
		newTable.Columns[k] = column.Clone()
	}
	return &newTable
}

// RandColumn rand column from table
func (t *Table) RandColumn() *Column {
	if len(t.Columns) == 0 {
		return nil
	}
	rdIndex := util.Rd(len(t.Columns))
	index := 0
	for _, column := range t.Columns {
		if rdIndex == index {
			return column.Clone()
		}
		index++
	}
	// should not reach here
	return nil
}

// GetColumns get ordered columns
func (t *Table) GetColumns() []*Column {
	var r []*Column
	for _, column := range t.Columns {
		r = append(r, column)
	}
	sort.Sort(byColumn(r))
	return r
}

// RandIndex rand indexes
func (t *Table) RandIndex() string {
	if len(t.Indexes) == 0 {
		return ""
	}
	return t.Indexes[util.Rd(len(t.Indexes))]
}

// AddToInnerTables Do NOT set InnerTableList directly
func (t *Table) AddToInnerTables(tables ...*Table) {
	t.InnerTableList = append(t.InnerTableList, tables...)

	// for removing duplicated items
	sort.Slice(t.InnerTableList, func(i, j int) bool {
		return strings.Compare(t.InnerTableList[i].Table, t.InnerTableList[j].Table) >= 0
	})
	tableList := make([]*Table, 0)
	for i := range t.InnerTableList {
		if i == 0 {
			tableList = append(tableList, t.InnerTableList[i])
			continue
		}
		if t.InnerTableList[i-1].Table == t.InnerTableList[i].Table {
			continue
		}
		tableList = append(tableList, t.InnerTableList[i])
	}
	t.InnerTableList = tableList
}
