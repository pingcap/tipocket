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
	"log"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/tests/sqllogictest"
)

var (
	sqllogicCaseURL = flag.String("p", "", "case url")
	testDir         = flag.String("d", "sqllogictest", "test case dir")
	taskCount       = flag.Int("t", 10, "concurrency")
	skipError       = flag.Bool("skip-error", false, "skip error for query test")
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

	provisioner, err := cluster.NewK8sProvisioner()
	if err != nil {
		log.Fatal(err)
	}
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: provisioner,
		ClientCreator: &sqllogictest.CaseCreator{
			Config: &sqllogictest.Config{
				SkipError: *skipError,
				TaskCount: 10,
				CaseURL:   *sqllogicCaseURL,
				TestDir:   *testDir,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		VerifySuit:  verify.Suit{},
		ClusterDefs: tidb.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace),
	}
	suit.Run(context.Background())
}
