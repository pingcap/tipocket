package rawkv

import (
	"github.com/pingcap/tipocket/db/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

// db is the TiDB database.
type db struct {
	cluster.Cluster
}

// Name returns the unique name for the database
func (db *db) Name() string {
	return "rawkv"
}

func init() {
	core.RegisterDB(&db{
		// RawKV does not use TiDB.
		cluster.Cluster{IncludeTidb: false},
	})
}
