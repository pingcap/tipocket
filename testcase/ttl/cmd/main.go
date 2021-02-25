package main

import (
	"context"
	"flag"

	"github.com/pingcap/tipocket/cmd/util"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	testcase "github.com/pingcap/tipocket/testcase/ttl"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

var (
	concurrency   = flag.Int("concurrency", 200, "concurrency worker count")
	dataPerWorker = flag.Int("data-per-worker", 2000, "data count for each concurrency worker")
)

func main() {
	flag.Parse()

	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}

	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: testcase.ClientCreator{
			Cfg: &testcase.Config{
				Concurrency:   *concurrency,
				DataPerWorker: *dataPerWorker,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
