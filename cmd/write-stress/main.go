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
	writestress "github.com/pingcap/tipocket/tests/write-stress"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

var (
	dataNum     = flag.Int("dataNum", 2000, "the number of data（the unit is 10 thoudstand)")
	concurrency = flag.Int("concurrency", 400, "concurrency of worker")
	batch       = flag.Int("batch", 100, "batch of insert sql")
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:        control.ModeSelfScheduled,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}
	//kvs := []string{"127.0.0.1:20160", "127.0.0.1:20162", "127.0.0.1:20161"}
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: cluster.NewK8sProvisioner(),
		//Provisioner: cluster.NewLocalClusterProvisioner([]string{"127.0.0.1:4000"}, []string{"127.0.0.1:2379"}, kvs),
		ClientCreator: writestress.ClientCreator{Cfg: &writestress.Config{
			DataNum:     *dataNum,
			Concurrency: *concurrency,
			Batch:       *batch,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
