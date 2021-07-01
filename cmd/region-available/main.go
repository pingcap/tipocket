package main

import (
	"context"
	"flag"
	"time"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	ra "github.com/pingcap/tipocket/tests/region-available"
)

var (
	dbName          = flag.String("db", "test", "database name")
	totalRows       = flag.Int("rows", 500000, "total rows")
	maxRespDuration = flag.Duration("max-resp-duration", 2*time.Second, "max response duration in seconds")
	concurrency     = flag.Int("concurrency", 1, "concurrency read worker count")
	sleepDuration   = flag.Duration("sleep-duration", 0, "sleep duration between two queries in milliseconds")
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
		ClientCreator: ra.ClientCreator{Cfg: &ra.Config{
			DBName:          *dbName,
			TotalRows:       *totalRows,
			Concurrency:     *concurrency,
			MaxResponseTime: *maxRespDuration,
			SleepDuration:   *sleepDuration,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.ClusterName,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
