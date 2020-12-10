package main

import (
	"context"
	"flag"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/verify"
	listappend "github.com/pingcap/tipocket/tests/list_append"
)

var (
	tableCount = flag.Int("table-count", 7, "Table count")
	readLock   = flag.String("read-lock", "FOR UPDATE", "Maybe empty or 'FOR UPDATE'")
	txnMode    = flag.String("txn-mode", "pessimistic", "Must be 'pessimistic', 'optimistic' or 'mixed'")
)

func main() {
	flag.Parse()

	suit := util.Suit{
		Config: &control.Config{
			Mode:         control.Mode(fixture.Context.Mode),
			ClientCount:  fixture.Context.ClientCount,
			RequestCount: fixture.Context.RequestCount,
			RunRound:     fixture.Context.RunRound,
			RunTime:      fixture.Context.RunTime,
			History:      fixture.Context.HistoryFile,
		},
		Provider:         cluster.NewDefaultClusterProvider(),
		ClientCreator:    listappend.NewClientCreator(*tableCount, *readLock, *txnMode, fixture.Context.ReplicaRead),
		NemesisGens:      util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClientRequestGen: util.OnClientLoop,
		VerifySuit: verify.Suit{
			Checker: listappend.AppendChecker{},
			Parser:  listappend.AppendParser{},
		},
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
