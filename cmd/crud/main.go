package main

import (
	"context"
	"flag"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/tests/crud"
)

var (
	dbName      = flag.String("db", "test", "database name")
	userCount   = flag.Int("user-count", 1000, "the number of users")
	postCount   = flag.Int("post-count", 1000, "the number of posts")
	updateUsers = flag.Int("update-users", 20, "the number of users updated")
	updatePosts = flag.Int("update-posts", 200, "the number of posts updated")
	interval    = flag.Duration("interval", 2*time.Second, "check interval")
	retryLimit  = flag.Int("retry-limit", 10, "retry count")
	txnMode     = flag.String("txn-mode", "pessimistic", "TiDB txn mode")
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

	provisioner, err := cluster.NewK8sProvisioner()
	if err != nil {
		log.Fatal(err)
	}
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: provisioner,
		ClientCreator: crud.CaseCreator{Cfg: &crud.Config{
			DBName:      *dbName,
			UserCount:   *userCount,
			PostCount:   *postCount,
			UpdateUsers: *updateUsers,
			UpdatePosts: *updatePosts,
			RetryLimit:  *retryLimit,
			Interval:    *interval,
			TxnMode:     *txnMode,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		VerifySuit:  verify.Suit{},
		ClusterDefs: tidb.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace),
	}
	suit.Run(context.Background())
}
