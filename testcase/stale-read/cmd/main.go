package main

import (
	"context"
	"flag"
	staleread "github.com/pingcap/tipocket/testcase/stale-read"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	logs "github.com/pingcap/tipocket/logsearch/pkg/logs"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
	}
	c := fixture.Context
	c.TiDBClusterConfig.TiDBReplicas = 1
	c.TiDBClusterConfig.TiKVReplicas = 5
	suit := util.Suit{
		Config:        &cfg,
		Provider:      cluster.NewDefaultClusterProvider(),
		ClientCreator: staleread.ClientCreator{},
		NemesisGens:   util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs:   test_infra.NewDefaultCluster(c.Namespace, c.ClusterName, c.TiDBClusterConfig),
		LogsClient:    logs.NewDiagnosticLogClient(),
	}
	suit.Run(context.Background())
}
