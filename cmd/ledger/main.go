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
	"time"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/tests/ledger"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

var (
	accounts    = flag.Int("accounts", 1000000, "the number of accounts")
	interval    = flag.Duration("interval", 2*time.Second, "check interval")
	concurrency = flag.Int("concurrency", 200, "concurrency of worker")
	txnMode     = flag.String("txn-mode", "pessimistic", "TiDB txn mode")
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
		ClientCreator: ledger.CaseCreator{Cfg: &ledger.Config{
			NumAccounts: *accounts,
			Concurrency: *concurrency,
			Interval:    *interval,
			TxnMode:     *txnMode,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		VerifySuit:  verify.Suit{},
		ClusterDefs: tidb.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace),
	}
	suit.Run(context.Background())
}
