package main

import (
	"context"
	"flag"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/check/porcupine"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	tidbInfra "github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
	rawkvlinearizability "github.com/pingcap/tipocket/tests/rawkv-linearizability"
)

func main() {
	flag.Parse()

	var checkers []core.Checker
	checkers = append(checkers, porcupine.Checker{})
	// TODO should add more checker

	verifySuit := verify.Suit{
		Model:   rawkvlinearizability.RawkvModel(),
		Checker: core.MultiChecker("rawkv-linearizability checkers", checkers...),
		Parser:  rawkvlinearizability.RawkvParser(),
	}
	cfg := control.Config{
		Mode:         control.Mode(fixture.Context.Mode),
		ClientCount:  fixture.Context.ClientCount,
		RequestCount: fixture.Context.RequestCount,
		RunRound:     fixture.Context.RunRound,
		RunTime:      fixture.Context.RunTime,
		History:      fixture.Context.HistoryFile,
	}
	//kvs := []string{"127.0.0.1:20160", "127.0.0.1:20162", "127.0.0.1:20161"}
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: cluster.NewK8sProvisioner(),
		//Provisioner:      cluster.NewLocalClusterProvisioner([]string{"127.0.0.1:4000"}, []string{"127.0.0.1:2379"}, kvs),
		ClientCreator:    rawkvlinearizability.RawkvClientCreator{},
		NemesisGens:      util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClientRequestGen: util.OnClientLoop,
		VerifySuit:       verifySuit,
		ClusterDefs:      tidbInfra.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace, fixture.Context.ImageVersion),
	}
	suit.Run(context.Background())
}
