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

	test_infra "github.com/pingcap/tipocket/pkg/test-infra"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/testcase/sqllogictest"
)

var (
	sqllogicCaseURL     = flag.String("p", "", "case url")
	testDir             = flag.String("d", "sqllogictest", "test case dir")
	taskCount           = flag.Int("t", 10, "concurrency")
	skipError           = flag.Bool("skip-error", false, "skip error for query test")
	replicaRead         = flag.String("tidb-replica-read", "leader", "tidb_replica_read mode, support values: leader / follower / leader-and-follower, default value: leader.")
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
	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: &sqllogictest.ClientCreator{
			Config: &sqllogictest.Config{
				SkipError:           *skipError,
				TaskCount:           *taskCount,
				CaseURL:             *sqllogicCaseURL,
				TestDir:             *testDir,
				ReplicaRead:         *replicaRead,
				DbName:              *dbname,
				TiFlashDataReplicas: *tiflashDataReplicas,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.ClusterName,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
