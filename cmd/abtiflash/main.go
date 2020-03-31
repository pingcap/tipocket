package main

import (
	"context"
	"flag"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/pocket/config"
	"github.com/pingcap/tipocket/pkg/pocket/creator"
	"github.com/pingcap/tipocket/pkg/test-infra/abtiflash"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

var (
	configPath = flag.String("config", "", "config file path")
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:        control.ModeSelfScheduled,
		ClientCount: 2,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}

	pocketConfig := config.Init()
	pocketConfig.Options.Serialize = true
	pocketConfig.Generator = config.Generator{SQLSmith: config.SQLSmith{
		TxnBegin:           20,
		TxnCommit:          20,
		TxnRollback:        10,
		DDLCreateTable:     0,
		DDLAlterTable:      0,
		DDLCreateIndex:     0,
		DMLSelect:          10,
		DMLSelectForUpdate: 30,
		DMLDelete:          10,
		DMLUpdate:          120,
		DMLInsert:          120,
		Sleep:              10,
	}}
	pocketConfig.Options.Path = fixture.Context.ABTestConfig.LogPath
	pocketConfig.Options.Concurrency = 1
	pocketConfig.Options.GeneralLog = fixture.Context.ABTestConfig.GeneralLog
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: cluster.NewK8sProvisioner(),
		ClientCreator: creator.PocketCreator{
			Config: creator.Config{
				ConfigPath: *configPath,
				Mode:       "abtiflash",
				Config:     pocketConfig,
			},
		},
		NemesisGens:      util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClientRequestGen: util.OnClientLoop,
		ClusterDefs: abtiflash.RecommendedCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.ImageVersion, fixture.Context.ABTestConfig.ClusterBVersion),
	}
	suit.Run(context.Background())
}
