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
	cdcbank "github.com/pingcap/tipocket/tests/cdc-bank"
)

var (
	accounts      = flag.Int("accounts", 50, "the number of accounts")
	concurrency   = flag.Int("concurrency", 32, "concurrency worker count")
	ticdcReplicas = flag.Int("ticdc-replicas", 1, "the ticdc replicas")
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}
	c := fixture.Context
	var waitWarmUpNemesisGens []core.NemesisGenerator
	for _, gen := range util.ParseNemesisGenerators(fixture.Context.Nemesis) {
		waitWarmUpNemesisGens = append(waitWarmUpNemesisGens, core.DelayNemesisGenerator{
			Gen:   gen,
			Delay: time.Minute * time.Duration(2),
		})
	}
	// Only set failpoints in the upstream
	downstreamCfg := c.TiDBClusterConfig
	downstreamCfg.TiDBFailPoint = ""
	c.TiDBClusterConfig.TiCDCReplicas = *ticdcReplicas
	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: cdcbank.ClientCreator{
			Cfg: &cdcbank.Config{
				Accounts:    *accounts,
				Concurrency: *concurrency,
				RunTime:     c.RunTime,
			},
		},
		NemesisGens:      waitWarmUpNemesisGens,
		ClientRequestGen: util.OnClientLoop,
		ClusterDefs:      test_infra.NewCDCCluster(c.Namespace, c.Namespace, c.TiDBClusterConfig, downstreamCfg),
	}
	suit.Run(context.Background())
}
