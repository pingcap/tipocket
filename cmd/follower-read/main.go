package main

import (
	"context"
	"flag"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/tests/follower"

	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
)

var (
	dbname      = flag.String("dbname", "test", "name of database to test")
	concurrency = flag.Int("concurrency", 200, "concurrency worker count")
)

func main() {
	cfg := control.Config{
		Mode:        control.ModeSelfScheduled,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}

	createFollowerReadCmd(&cfg)
}

func createFollowerReadCmd(cfg *control.Config) {
	suit := util.Suit{
		Config: cfg,
		ClientCreator: follower.ClientCreator{
			Cfg: &follower.Config{
				DBName:      *dbname,
				Concurrency: *concurrency,
			},
		},
		Provisioner: cluster.NewK8sProvisioner(),
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
