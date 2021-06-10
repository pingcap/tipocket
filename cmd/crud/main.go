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
	"github.com/pingcap/tipocket/tests/crud"
)

var (
	dbName              = flag.String("db", "test", "database name")
	userCount           = flag.Int("user-count", 1000, "the number of users")
	postCount           = flag.Int("post-count", 1000, "the number of posts")
	updateUsers         = flag.Int("update-users", 20, "the number of users updated")
	updatePosts         = flag.Int("update-posts", 200, "the number of posts updated")
	interval            = flag.Duration("interval", 2*time.Second, "check interval")
	retryLimit          = flag.Int("retry-limit", 10, "retry count")
	txnMode             = flag.String("txn-mode", "pessimistic", "TiDB txn mode")
	tiflashDataReplicas = flag.Int("tiflash-data-replicas", 0, "the number of the tiflash data replica")
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
		ClientCreator: crud.ClientCreator{Cfg: &crud.Config{
			DBName:              *dbName,
			UserCount:           *userCount,
			PostCount:           *postCount,
			UpdateUsers:         *updateUsers,
			UpdatePosts:         *updatePosts,
			RetryLimit:          *retryLimit,
			Interval:            *interval,
			TxnMode:             *txnMode,
			TiFlashDataReplicas: *tiflashDataReplicas,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.ClusterName,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
