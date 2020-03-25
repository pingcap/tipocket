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

	"github.com/pingcap/go-tpc/tpcc"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/db/tidb"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	tidbInfra "github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
)

var (
	ticker      = flag.Duration("ticker", time.Second, "ticker control request emitting freq")
	qosFile     = flag.String("qos-file", "./qos.log", "qos file")
	checkerName = flag.String("checker", "consistency", "consistency or qos")
	warehouses  = flag.Int("warehouses", 10, "tpcc warehouses")
	threads     = flag.Int("T", 10, "tpcc terminals")
)

func main() {
	util.PrintInfo()
	flag.Parse()

	cfg := control.Config{
		Mode:         control.ModeSequential,
		ClientCount:  fixture.Context.ClientCount,
		RequestCount: fixture.Context.RequestCount,
		RunRound:     fixture.Context.RunRound,
		RunTime:      fixture.Context.RunTime,
		History:      fixture.Context.HistoryFile,
	}
	clientCreator := &tidb.TPCCClientCreator{
		Config: &tpcc.Config{
			Threads:    *threads,
			Warehouses: *warehouses,
			Parts:      1,
		},
	}
	creator := clientCreator
	var checker core.Checker
	switch *checkerName {
	case "consistency":
		checker = &tidb.TPCCChecker{CreatorRef: clientCreator}
	case "qos":
		checker = core.MultiChecker("tpcc qos", tidb.TPCCQosChecker(time.Minute, *qosFile), &tidb.TPCCChecker{CreatorRef: clientCreator})
	}
	verifySuit := verify.Suit{
		Model:   nil,
		Checker: checker,
		Parser:  tidb.TPCCParser(),
	}
	var waitWarmUpNemesisGens []core.NemesisGenerator
	for _, gen := range util.ParseNemesisGenerators(fixture.Context.Nemesis) {
		waitWarmUpNemesisGens = append(waitWarmUpNemesisGens, core.DelayNemesisGenerator{
			Gen:   gen,
			Delay: time.Minute * time.Duration(2),
		})
	}
	suit := util.Suit{
		Config:           &cfg,
		Provisioner:      cluster.NewK8sProvisioner(),
		ClientCreator:    creator,
		NemesisGens:      waitWarmUpNemesisGens,
		ClientRequestGen: util.BuildClientLoopThrottle(*ticker),
		VerifySuit:       verifySuit,
		ClusterDefs:      tidbInfra.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace),
	}
	suit.Run(context.Background())
}
