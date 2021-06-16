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

// Note: This part of the code is port from https://github.com/pingcap/schrodinger-test/tree/master/transaction/bank .
//  And it changes schrodinger-test to kubernetes-style tests.

package main

import (
	"context"
	"flag"
	"time"

	"github.com/pingcap/tipocket/cmd/util"
	logs "github.com/pingcap/tipocket/logsearch/pkg/logs"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/testcase/tiflashu1"
)

var (
	// case config
	updateConcurrency   = flag.Int("update-concurrency", 100, "concurrency update worker count")
	verifyConcurrency   = flag.Int("verify-concurrency", 10, "concurrency verify worker count")
	initialRecordNum    = flag.Int("initial-record-num", 600000, "initial row number")
	insertBatchCount    = flag.Int("insert-batch-count", 50, "insert batch count")
	interval            = flag.Duration("interval", 1*time.Second, "the interval")
	dbname              = flag.String("dbname", "test", "name of database to test")
	tiflashDataReplicas = flag.Int("tiflash-data-replicas", 0, "the number of the tiflash data replica")
)

func main() {
	flag.Parse()

	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}

	tiflashu1Config := tiflashu1.Config{
		UpdateConcurrency:   *updateConcurrency,
		VerifyConcurrency:   *verifyConcurrency,
		InitialRecordNum:    *initialRecordNum,
		InsertBatchCount:    *insertBatchCount,
		Interval:            *interval,
		DbName:              *dbname,
		TiFlashDataReplicas: *tiflashDataReplicas,
	}

	suit := util.Suit{
		Config:        &cfg,
		Provider:      cluster.NewDefaultClusterProvider(),
		ClientCreator: tiflashu1.ClientCreator{Cfg: &tiflashu1Config},
		NemesisGens:   util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.ClusterName,
			fixture.Context.TiDBClusterConfig),
		LogsClient: logs.NewDiagnosticLogClient(),
	}
	suit.Run(context.Background())
}
