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
	tableCount = flag.Int("table-count", 1, "table count")
	readLock   = flag.String("readLock", "FOR UPDATE", "maybe empty or 'FOR UPDATE'")
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
		Provisioner:      cluster.NewK8sProvisioner(),
		ClientCreator:    listappend.NewClientCreator(*tableCount, *readLock),
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
