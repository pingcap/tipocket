package sqlsmith

import "testing"

func TestSQLSmith_DataGenerator(t *testing.T) {
	ss := New()

	ss.LoadSchema(schema, indexes)

	ss.SetDB(dbname)

	gen, _ := ss.GenData(10, 5)

	for sqls := gen.Next(); len(sqls) != 0; sqls = gen.Next() {
		t.Log(sqls)
	}
	// t.Log("rd string", ss.randString(ss.rd(100)))
}
