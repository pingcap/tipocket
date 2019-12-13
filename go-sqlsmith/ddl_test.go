package sqlsmith

import "testing"

func TestSQLSmith_CreateTable(t *testing.T) {
	ss := New()
	ss.LoadSchema([][5]string{}, make(map[string][]string))
	ss.SetDB(dbname)
	sql, _ := ss.CreateTableStmt()
	t.Log(sql)
}

func TestSQLSmith_AlterTable(t *testing.T) {
	ss := New()
	indexes["users"] = []string{"idx1", "idx2"}
	ss.LoadSchema(schema, indexes)
	ss.SetDB(dbname)

	sql, _ := ss.AlterTableStmt()
	t.Log(sql)
}

func TestSQLSmith_CreateIndex(t *testing.T) {
	ss := New()
	ss.LoadSchema(schema, indexes)
	ss.SetDB(dbname)

	sql, _ := ss.CreateIndexStmt()
	t.Log(sql)
}
