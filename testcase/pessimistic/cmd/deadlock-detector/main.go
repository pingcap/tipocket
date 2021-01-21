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
	dlc "github.com/pingcap/tipocket/testcase/pessimistic/pkg/deadlock"
)

var (
	dbName           = flag.String("db", "test", "database name")
	tableNum         = flag.Int("table-num", 10, "the table num of single statement rollback case")
	deadlockInterval = flag.Duration("deadlock-interval", 100*time.Millisecond, "the interval of detecting deadlock")
	deadlockTimeout  = flag.Duration("deadlock-duration", 10*time.Second, "the max duration of detecting deadlock")

	// Uncomment this when we can specify nemesis duration
	//shuffleInterval  = flag.Duration("shuffle-interval", 10*time.Second, "the interval of shuffling the leader of deadlock detector")
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
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: dlc.ClientCreator{Cfg: &dlc.Config{
			DBName:           *dbName,
			TableNum:         *tableNum,
			DeadlockInterval: *deadlockInterval,
			DeadlockTimeout:  *deadlockTimeout,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
