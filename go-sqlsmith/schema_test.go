package sqlsmith

import (
	"testing"
	// "github.com/stretchr/testify/assert"
)

func TestSQLSmith_Schema_TableMerge(t *testing.T) {
	// ss := New()
	// ss.LoadSchema([][5]string{
	// 	{"test", "balances", "BASE TABLE", "id", "int(11)"},
	// 	{"test", "balances", "BASE TABLE", "user", "varchar(255)"},
	// 	{"test", "balances", "BASE TABLE", "money", "int(11)"},
	// 	{"test", "records", "BASE TABLE", "id", "int(11)"},
	// 	{"test", "records", "BASE TABLE", "from_id", "int(11)"},
	// 	{"test", "records", "BASE TABLE", "to_id", "int(11)"},
	// 	{"test", "records", "BASE TABLE", "created_at", "timestamp"},
	// })

	
	// table, onColumns := ss.mergeTable(ss.Databases["test"].Tables["balances"], ss.Databases["test"].Tables["records"])
	// assert.Equal(t, len(onColumns), 2)
	// assert.Equal(t, onColumns[0].Table, "balances")
	// assert.Equal(t, onColumns[0].Column, "id")
	// assert.Equal(t, onColumns[1].Table, "records")
	// assert.Equal(t, onColumns[1].Column, "id")
	// assert.Equal(t, len(table.Columns), 7)
}
