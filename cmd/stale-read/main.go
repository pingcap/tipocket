package main

import (
	"context"
	"flag"
	"time"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	staleread "github.com/pingcap/tipocket/tests/stale-read"

	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
)

var (
	dbname          = flag.String("dbname", "test", "name of database to test")
	concurrency     = flag.Int("concurrency", 200, "concurrency worker count")
	totalRows       = flag.Int("rows", 3000000, "total rows of data")
	requestInterval = flag.Duration("request-interval", 50*time.Millisecond, "request freq")
	maxStaleness    = flag.Int("staleness", 100, "the max staleness in second")
)

func main() {
	flag.Parse()

	cfg := control.Config{
		Mode:        control.ModeSelfScheduled,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}

	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: &staleread.ClientCreator{
			Cfg: &staleread.Config{
				DBName:          *dbname,
				Concurrency:     *concurrency,
				TotalRows:       *totalRows,
				RequestInterval: *requestInterval,
				MaxStaleness:    *maxStaleness,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
