package types

import (
	"sort"
	"github.com/pingcap/tipocket/go-sqlsmith/util"
)

// Table defines database table
type Table struct {
	DB string
	Table string
	OriginTable string
	Type string
	Columns map[string]*Column
	Indexes []string
}

type byColumn []*Column
func (a byColumn) Len() int           { return len(a) }
func (a byColumn) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
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
	newTable := *t
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
