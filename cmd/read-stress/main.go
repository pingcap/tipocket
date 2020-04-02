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
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	readstress "github.com/pingcap/tipocket/tests/read-stress"

	// use mysql
	_ "github.com/go-sql-driver/mysql"
)

var (
	numRows          = flag.Int("rows", 10000000, "the number of rows")
	smallConcurrency = flag.Int("small-concurrency", 32, "concurrency of small queries")
	smallTimeout     = flag.Duration("small-timeout", 100*time.Millisecond, "maximum latency of small queries")
	largeConcurrency = flag.Int("big-concurrency", 1, "concurrency of large queries")
	largeTimeout     = flag.Duration("large-timeout", 10*time.Second, "maximum latency of big queries")
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
		ClientCreator: readstress.CaseCreator{
			NumRows:          *numRows,
			SmallConcurrency: *smallConcurrency,
			SmallTimeout:     *smallTimeout,
			LargeConcurrency: *largeConcurrency,
			LargeTimeout:     *largeTimeout,
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: tidb.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace, fixture.Context.ImageVersion, fixture.TiDBImageConfig{}),
	}
	suit.Run(context.Background())
}
