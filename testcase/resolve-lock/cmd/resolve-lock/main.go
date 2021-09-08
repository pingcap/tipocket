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

	logs "github.com/pingcap/tipocket/logsearch/pkg/logs"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"

	resolvelock "github.com/pingcap/tipocket/testcase/resolve-lock"
)

var (
	enableGreenGC = flag.Bool("enable-green-gc", true, "whether to enable green gc")
	regionCount   = flag.Int("region-count", 200, "count of regions")
	lockPerRegion = flag.Int("lock-per-region", 10, "count of locks in each region")
	workers       = flag.Int("worker", 10, "count of workers to generate locks")
	localMode     = flag.Bool("local-mode", false, "run the test in local mode which means use localhost cluster to run the test")
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}
	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: resolvelock.CaseCreator{Cfg: &resolvelock.Config{
			EnableGreenGC: *enableGreenGC,
			RegionCount:   *regionCount,
			LockPerRegion: *lockPerRegion,
			Worker:        *workers,
			LocalMode:     *localMode,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.ClusterName,
			fixture.Context.TiDBClusterConfig),
		LogsClient: logs.NewDiagnosticLogClient(),
	}
	suit.Run(context.Background())
}
