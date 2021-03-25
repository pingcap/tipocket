// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"flag"
	"time"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/config"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/creator"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/types"
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
	pocketConfig.Options.Serialize = false
	pocketConfig.Options.Path = fixture.Context.DMConfig.LogPath
	pocketConfig.Options.Concurrency = 1
	pocketConfig.Options.CheckDuration = types.Duration{
		Duration: 10 * time.Second, // less data in one check round to ensure downstream can catchup upstream.
	}

	var waitWarmUpNemesisGens []core.NemesisGenerator
	for _, gen := range util.ParseNemesisGenerators(fixture.Context.Nemesis) {
		waitWarmUpNemesisGens = append(waitWarmUpNemesisGens, core.DelayNemesisGenerator{
			Gen:   gen,
			Delay: time.Minute * time.Duration(2),
		})
	}

	c := fixture.Context
	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: creator.PocketCreator{
			Config: creator.Config{
				ConfigPath: *configPath,
				Mode:       "dm",
				Config:     pocketConfig,
			},
		},
		NemesisGens:      waitWarmUpNemesisGens,
		ClientRequestGen: util.OnClientLoop,
		ClusterDefs:      test_infra.NewDMCluster(c.Namespace, c.ClusterName, c.DMConfig, c.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
