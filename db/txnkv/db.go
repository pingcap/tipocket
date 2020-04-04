package txnkv

//
//import (
//	"github.com/pingcap/tipocket/db/cluster"
//	"github.com/pingcap/tipocket/pkg/core"
//)
//
//// db is the transactional KV database.
//type db struct {
//	cluster.Cluster
//}
//
//// Name returns the unique name for the database
//func (db *db) Name() string {
//	return "txnkv"
//}
//
//func init() {
//	core.RegisterDB(&db{
//		// TxnKV does not use TiDB.
//		cluster.Cluster{IncludeTidb: false},
//	})
//}
