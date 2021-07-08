// Copyright 2021 PingCAP, Inc.
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
	staleread "github.com/pingcap/tipocket/testcase/stale-read"

	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
)

var (
	dbname          = flag.String("dbname", "test", "name of database to test")
	concurrency     = flag.Int("concurrency", 200, "concurrency worker count")
	totalRows       = flag.Int("rows", 4000000, "total rows of data")
	requestInterval = flag.Duration("request-interval", 25*time.Millisecond, "request freq")
	maxStaleness    = flag.Int("staleness", 100, "the max staleness in second")
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
		ClientCreator: &staleread.ClientCreator{
			Cfg: &staleread.Config{
				DBName:          *dbname,
				Concurrency:     *concurrency,
				TotalRows:       *totalRows,
				RequestInterval: *requestInterval,
				MaxStaleness:    *maxStaleness,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
