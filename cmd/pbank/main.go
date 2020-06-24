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
	"strings"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/db/tidb"
	adminChecker "github.com/pingcap/tipocket/pkg/check/admin-check"
	"github.com/pingcap/tipocket/pkg/check/porcupine"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/verify"
)

var (
	clientCase   = flag.String("case", "bank", "client test case, like bank,multi_bank")
	checkerNames = flag.String("checkers", "porcupine", "checker names, separate by comma. eg, porcupine,admin_check")
)

func main() {
	flag.Parse()

	var (
		creator core.ClientCreator
		parser  = tidb.BankParser()
		model   = tidb.BankModel()
		cfg     = control.Config{
			Mode:         control.Mode(fixture.Context.Mode),
			ClientCount:  fixture.Context.ClientCount,
			RequestCount: fixture.Context.RequestCount,
			RunRound:     fixture.Context.RunRound,
			RunTime:      fixture.Context.RunTime,
			History:      fixture.Context.HistoryFile,
		}
	)
	switch *clientCase {
	case "bank":
		creator = tidb.BankClientCreator{}
	case "multi_bank":
		creator = tidb.MultiBankClientCreator{}
	case "long_fork":
		creator = tidb.LongForkClientCreator{}
	//case "sequential":
	//	creator = tidb.SequentialClientCreator{}
	default:
		log.Fatalf("invalid client test case %s", *clientCase)
	}

	names := strings.Split(*checkerNames, ",")
	var checkers []core.Checker
	for _, name := range names {
		switch name {
		case "porcupine":
			checkers = append(checkers, porcupine.Checker{})
		case "tidb_bank_tso":
			checkers = append(checkers, tidb.BankTsoChecker())
		case "long_fork_checker":
			checkers = append(checkers, tidb.LongForkChecker())
			parser = tidb.LongForkParser()
			model = nil
		case "admin_check":
			checkers = append(checkers, adminChecker.AdminChecker(&cfg))
		//case "sequential_checker":
		//	checker = tidb.NewSequentialChecker()
		//	parser = tidb.NewSequentialParser()
		//	model = nil
		default:
			log.Fatalf("invalid checker %s", *checkerNames)
		}
	}

	verifySuit := verify.Suit{
		Model:   model,
		Checker: core.MultiChecker("tidb checkers", checkers...),
		Parser:  parser,
	}
	suit := util.Suit{
		Config:           &cfg,
		Provider:         cluster.NewDefaultClusterProvider(),
		ClientCreator:    creator,
		NemesisGens:      util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClientRequestGen: util.OnClientLoop,
		VerifySuit:       verifySuit,
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
