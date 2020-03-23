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
	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/pkg/test-infra/tidb"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	resolvelock "github.com/pingcap/tipocket/tests/resolve-lock"
)

var (
	concurrency   = flag.Int("concurrency", 200, "concurrency of worker")
	enableGreenGC = flag.Bool("enable-green-gc", true, "whether to enable green gc")
	tableSize     = flag.Int("table-size", 1000000, "the size of the table")
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:        control.ModeSelfScheduled,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: cluster.NewK8sProvisioner(),
		ClientCreator: resolvelock.CaseCreator{Cfg: &resolvelock.Config{
			Concurrency:             *concurrency,
			EnableGreenGC:           * enableGreenGC,
			TableSize:               *tableSize,
			LocksPerRegion:          1,
			GenerateLockConcurrency: 16,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: tidb.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace),
	}
	suit.Run(context.Background())
}
