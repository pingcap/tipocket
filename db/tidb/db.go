package tidb

import (
	"github.com/pingcap/tipocket/db/cluster"
)

// db is the TiDB database.
type db struct {
	cluster.Cluster
}

// Name returns the unique name for the database
func (db *db) Name() string {
	return "tidb"
}

//func init() {
//	core.RegisterDB(&db{
//		cluster.Cluster{IncludeTidb: true},
//	})
//}
