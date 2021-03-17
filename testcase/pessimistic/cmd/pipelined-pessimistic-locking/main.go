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
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/testcase/pessimistic/pkg/pipeline"
)

var (
	tableSize        = flag.Int("table-size", 50, "row counts in the table")
	contenderCount   = flag.Int("contenders", 2, "the count of sessions updating the same row")
	scheduleInterval = flag.Duration("schedule-interval", 3*time.Second, "periodically schedule regions of the table with a specified interval")
	reportInterval   = flag.Duration("report-interval", 10*time.Second, "periodically report intermediate statistics with a specified interval. 0 disables intermediate reports [0]")
	strict           = flag.Bool("strict", false, "strict means as long as one transaction fails to commit, the test quits")
	localMode        = flag.Bool("local-mode", false, "run the test in local mode which means use localhost cluster to run the test")
)

// This test tests the success rate of pipelined pessimistic locking.
//
// A pessimistic transaction using pipelined pessimistic locking can fail to commit when
//   1. Fails to write pessimistic locks due to region error;
//   2. And fails to amend pessimistic locks when prewrites if forUpdateTs is updated due to write conflict.
// This test create a table with tableSize rows. Each row has contenderCount pessimistic transactions updating it concurrently and
// schedule regions of the table so that pipelined pessimistic lock can fail.
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
		ClientCreator: &pipeline.Config{
			TableSize:        *tableSize,
			ContenderCount:   *contenderCount,
			ScheduleInterval: *scheduleInterval,
			ReportInterval:   *reportInterval,
			Strict:           *strict,
			LocalMode:        *localMode,
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.ClusterName,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
