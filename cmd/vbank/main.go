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
	"github.com/pingcap/tipocket/pkg/check/porcupine"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	tidbInfra "github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/tests/vbank"
)

var (
	pkType        = flag.String("pk_type", "int", "primary key type, int,decimal,string")
	partition     = flag.Bool("partition", true, "use partitioned table")
	useRange      = flag.Bool("range", false, "use range condition")
	updateInPlace = flag.Bool("update_in_place", false, "use update in place mode")
	readCommitted = flag.Bool("read_committed", false, "use READ-COMMITTED isolation level")
	local         = flag.Bool("local", false, "use local cluster")
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:         control.Mode(fixture.Context.Mode),
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
	suit := util.Suit{
		Config:           &cfg,
		ClientCreator:    &vbank.ClientCreator{},
		ClientRequestGen: util.OnClientLoop,
		VerifySuit:       verifySuit,
	}
	if *local {
		suit.Provisioner = cluster.NewLocalClusterProvisioner([]string{"127.0.0.1:4000"}, nil, nil)
	} else {
		suit.Provisioner = cluster.NewK8sProvisioner()
		suit.NemesisGens = util.ParseNemesisGenerators(fixture.Context.Nemesis)
		suit.ClusterDefs = tidbInfra.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace, fixture.Context.ImageVersion)
	}
	vbCfg := &vbank.Config{
		PKType:        *pkType,
		Partition:     *partition,
		Range:         *useRange,
		ReadCommitted: *readCommitted,
		UpdateInPlace: *updateInPlace,
	}
	ctx := context.WithValue(context.Background(), vbank.ConfigKey, vbCfg)
	suit.Run(ctx)
}
