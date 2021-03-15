package main

import (
	"context"
	"flag"

	"github.com/pingcap/tipocket/cmd/util"
	logs "github.com/pingcap/tipocket/logsearch/pkg/logs"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/verify"
	rwregister "github.com/pingcap/tipocket/testcase/rw-register"
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
			Mode:         control.ModeOnSchedule,
			ClientCount:  fixture.Context.ClientCount,
			RequestCount: fixture.Context.RequestCount,
			RunRound:     fixture.Context.RunRound,
			RunTime:      fixture.Context.RunTime,
			History:      fixture.Context.HistoryFile,
		},
		Provider:         cluster.NewDefaultClusterProvider(),
		ClientCreator:    rwregister.NewClientCreator(*tableCount, *readLock, *txnMode, fixture.Context.ReplicaRead),
		NemesisGens:      util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClientRequestGen: util.OnClientLoop,
		VerifySuit: verify.Suit{
			Checker: rwregister.RegisterChecker{},
			Parser:  rwregister.RegisterParser{},
		},
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
		LogsClient: logs.NewDiagnosticLogClient(),
	}
	suit.Run(context.Background())
}
