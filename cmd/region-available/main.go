package main

import (
	"context"
	"flag"
	"time"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
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
		Mode:        control.ModeSelfScheduled,
		ClientCount: 1,
		DB:          "noop",
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: cluster.NewK8sProvisioner(),
		ClientCreator: ra.CaseCreator{Cfg: &ra.Config{
			DBName:          *dbName,
			TotalRows:       *totalRows,
			Concurrency:     *concurrency,
			MaxResponseTime: *maxRespDuration,
			SleepDuration:   *sleepDuration,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		VerifySuit:  verify.Suit{},
		ClusterDefs: tidb.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace),
	}
	suit.Run(context.Background())
}
