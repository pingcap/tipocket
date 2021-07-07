package main

import (
	"context"
	"flag"
	"time"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	logs "github.com/pingcap/tipocket/logsearch/pkg/logs"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	staleread "github.com/pingcap/tipocket/testcase/stale-read"
)

var (
	workerCount    = flag.Int("sysbench-worker-count", 1, "the worker count of the sysbench")
	runDuration    = flag.Duration("sysbench-duration", 1*time.Minute, "the duration of the sysbench running")
	rowsEachInsert = flag.Int("rows-each-insert", 50, "rows each time insert")
	insertCount    = flag.Int("insert-count", 20, "count of the inserting")
	preSecs        = flag.Int("pre-secs", 3, "previous seconds of stale read query")
)

func main() {
	flag.Parse()
	fixture.Context.ClusterName = "stale-read"
	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
	}
	c := fixture.Context
	c.TiDBClusterConfig.PDReplicas = 1
	c.TiDBClusterConfig.TiDBReplicas = 1
	c.TiDBClusterConfig.TiKVReplicas = 3
	c.TiDBClusterConfig.LogStorageClassName = "shared-sas-disks"
	c.TiDBClusterConfig.PDStorageClassName = "shared-nvme-disks"
	c.TiDBClusterConfig.TiKVStorageClassName = "nvme-disks"
	c.TiDBClusterConfig.PDImage = "hub.pingcap.net/gaosong/pd:newly-master"
	c.TiDBClusterConfig.TiDBImage = "hub.pingcap.net/gaosong/tidb:newly-master"
	c.TiDBClusterConfig.TiKVImage = "hub.pingcap.net/gaosong/tikv:newly-master"
	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: staleread.ClientCreator{
			Config: staleread.Config{
				SysBenchWorkerCount: *workerCount,
				SysBenchDuration:    *runDuration,
				RowsEachInsert:      *rowsEachInsert,
				InsertCount:         *insertCount,
				PreSec:              *preSecs,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(c.Namespace, c.ClusterName, c.TiDBClusterConfig),
		LogsClient:  logs.NewDiagnosticLogClient(),
	}
	suit.Run(context.Background())
}
