package sqlsmith

import "testing"

func TestSQLSmith_GenData(t *testing.T) {
	ss := New()
	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)
	sqls, _ := ss.BatchData(2, 2)
	for _, sql := range sqls {
		t.Log(sql)
	}
	// t.Log("rd string", ss.randString(ss.rd(100)))
}
