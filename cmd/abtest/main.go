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

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/pocket/config"
	"github.com/pingcap/tipocket/pkg/pocket/creator"
	"github.com/pingcap/tipocket/pkg/test-infra/abtest"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/verify"
)

var (
	configPath = flag.String("config", "", "config file path")
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

	verifySuit := verify.Suit{
		Model:   &core.NoopModel{},
		Checker: core.NoopChecker{},
		Parser:  nil,
	}

	pocketConfig := config.Init()
	pocketConfig.Options.Serialize = true
	pocketConfig.Options.Path = fixture.Context.ABTestConfig.LogPath
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: cluster.NewK8sProvisioner(),
		ClientCreator: creator.PocketCreator{
			Config: creator.Config{
				ConfigPath: *configPath,
				Mode:       "abtest",
				Config:     pocketConfig,
			},
		},
		NemesisGens:      util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClientRequestGen: util.OnClientLoop,
		VerifySuit:       verifySuit,
		ClusterDefs:      abtest.RecommendedCluster(fixture.Context.Namespace, fixture.Context.Namespace),
	}
	suit.Run(context.Background())
}
