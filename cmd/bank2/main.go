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

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/tests/bank2"
)

var (
	accounts    = flag.Int("accounts", 1000000, "the number of accounts")
	interval    = flag.Duration("interval", 2*time.Second, "the interval")
	tables      = flag.Int("tables", 1, "the number of the tables")
	concurrency = flag.Int("concurrency", 200, "concurrency worker count")
	retryLimit  = flag.Int("retry-limit", 200, "retry count")
	longTxn     = flag.Bool("long-txn", true, "enable long-term transactions")
	runMode     = flag.String("run-mode", "online", "case mode, support values: online / dev, default value: online")
	contention  = flag.String("contention", "low", "contention level, support values: high / low, default value: low")
	pessimistic = flag.Bool("pessimistic", false, "use pessimistic transaction")
	minLength   = flag.Int("min-value-length", 0, "minimum value inserted into rocksdb")
	maxLength   = flag.Int("max-value-length", 128, "maximum value inserted into rocksdb")
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
		ClientCreator: bank2.CaseCreator{Cfg: &bank2.Config{
			NumAccounts:   *accounts,
			Interval:      *interval,
			TableNum:      *tables,
			Concurrency:   *concurrency,
			RetryLimit:    *retryLimit,
			RunMode:       *runMode,
			MinLength:     *minLength,
			MaxLength:     *maxLength,
			EnableLongTxn: *longTxn,
			Contention:    *contention,
			Pessimistic:   *pessimistic,
		}},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		VerifySuit:  verify.Suit{},
		ClusterDefs: tidb.RecommendedTiDBCluster(fixture.Context.Namespace, fixture.Context.Namespace, fixture.Context.ImageVersion),
	}
	suit.Run(context.Background())
}
