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

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/tests/ondup"
)

var (
	dbName     = flag.String("db", "test", "database name")
	numRows    = flag.Int("num-rows", 10000, "number of rows")
	retryLimit = flag.Int("retry-limit", 2, "retry count")
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
		ClientCreator: ondup.CaseCreator{Cfg: &ondup.Config{
			DBName:     *dbName,
			NumRows:    *numRows,
			RetryLimit: *retryLimit,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: tidb.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace, fixture.Context.ImageVersion),
	}
	suit.Run(context.Background())
}
