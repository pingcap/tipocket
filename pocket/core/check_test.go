package core

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestMakeCompareSQLs(t *testing.T) {
	schema := [][5]string{
		{"test", "balances", "BASE TABLE", "id", "int(11)"},
		{"test", "balances", "BASE TABLE", "user", "varchar(255)"},
		{"test", "balances", "BASE TABLE", "money", "int(11)"},
		{"test", "records", "BASE TABLE", "id", "int(11)"},
		{"test", "records", "BASE TABLE", "from_id", "int(11)"},
		{"test", "records", "BASE TABLE", "to_id", "int(11)"},
		{"test", "records", "BASE TABLE", "created_at", "timestamp"},
	}

	sqls := makeCompareSQLs(schema)
	assert.Equal(t, sqls[0], "SELECT COUNT(1) FROM balances")
	assert.Equal(t, sqls[1], "SELECT COUNT(1) FROM records")
	assert.Equal(t, sqls[2], "SELECT from_id, to_id, created_at FROM records ORDER BY from_id, to_id, created_at")
	assert.Equal(t, sqls[3], "SELECT user, money FROM balances ORDER BY user, money")
}
