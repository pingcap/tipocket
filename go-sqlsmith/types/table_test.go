package types

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestSQLSmith_TableGetColumns(t *testing.T) {
	table := Table{
		Columns: make(map[string]*Column),
	}
	table.Columns["id"] = &Column{Column:"id"}
	table.Columns["a"] = &Column{Column:"a"}
	table.Columns["c"] = &Column{Column:"c"}
	table.Columns["b"] = &Column{Column:"b"}

	columns := table.GetColumns()
	assert.Equal(t, columns[0].Column, "a")
	assert.Equal(t, columns[1].Column, "b")
	assert.Equal(t, columns[2].Column, "c")
	assert.Equal(t, columns[3].Column, "id")
}
