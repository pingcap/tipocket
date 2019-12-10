package sqlsmith

import "testing"

func TestSQLSmith_ToSQL(t *testing.T) {
	ss := New()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)

	ss.SetDB("community")
	sql, _ := ss.SelectStmt(3)
	t.Log(sql)
}
