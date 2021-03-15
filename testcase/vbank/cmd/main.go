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

	"github.com/pingcap/tipocket/cmd/util"
	logs "github.com/pingcap/tipocket/logsearch/pkg/logs"
	"github.com/pingcap/tipocket/pkg/check/porcupine"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/testcase/vbank"
)

var (
	clusterName = flag.String("cluster-name", "", "tidb cluster name (default to namespace)")

	pkType        = flag.String("pk_type", "int", "primary key type, int,decimal,string")
	partition     = flag.Bool("partition", true, "use partitioned table")
	useRange      = flag.Bool("range", false, "use range condition")
	updateInPlace = flag.Bool("update_in_place", false, "use update in place mode")
	readCommitted = flag.Bool("read_committed", false, "use READ-COMMITTED isolation level")
	connParams    = flag.String("conn_params", "", "connection parameters")
)

func main() {
	flag.Parse()
	if len(*clusterName) == 0 {
		*clusterName = fixture.Context.Namespace
	}
	cfg := control.Config{
		Mode:         control.ModeOnSchedule,
		ClientCount:  fixture.Context.ClientCount,
		RequestCount: fixture.Context.RequestCount,
		RunRound:     fixture.Context.RunRound,
		RunTime:      fixture.Context.RunTime,
		History:      fixture.Context.HistoryFile,
	}
	verifySuit := verify.Suit{
		Model:   &vbank.Model{},
		Checker: core.MultiChecker("v_bank checkers", porcupine.Checker{}),
		Parser:  &vbank.Parser{},
	}
	vbCfg := &vbank.Config{
		PKType:        *pkType,
		Partition:     *partition,
		Range:         *useRange,
		ReadCommitted: *readCommitted,
		UpdateInPlace: *updateInPlace,
		ConnParams:    *connParams,
	}
	suit := util.Suit{
		Config:           &cfg,
		ClientCreator:    vbank.NewClientCreator(vbCfg),
		ClientRequestGen: util.OnClientLoop,
		VerifySuit:       verifySuit,
		Provider:         cluster.NewDefaultClusterProvider(),
		NemesisGens:      util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs:      test_infra.NewDefaultCluster(fixture.Context.Namespace, *clusterName, fixture.Context.TiDBClusterConfig),
		LogsClient:       logs.NewDiagnosticLogClient(),
	}
	suit.Run(context.Background())
}
