package main

import (
	"context"
	"flag"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/config"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/creator"
)

var (
	configPath = flag.String("config", "", "config file path")
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
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
	c := fixture.Context
	pocketConfig.Options.Path = "abtest.log"
	pocketConfig.Options.Concurrency = 1
	pocketConfig.Options.GeneralLog = fixture.Context.ABTestConfig.GeneralLog
	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: creator.PocketCreator{
			Config: creator.Config{
				ConfigPath: *configPath,
				Mode:       "tiflash-abtest",
				Config:     pocketConfig,
			},
		},
		NemesisGens:      util.ParseNemesisGenerators(c.Nemesis),
		ClientRequestGen: util.OnClientLoop,
		ClusterDefs:      test_infra.NewTiFlashABTestCluster(c.Namespace, c.Namespace, c.TiDBClusterConfig, c.ABTestConfig.ClusterBConfig),
	}
	suit.Run(context.Background())
}
