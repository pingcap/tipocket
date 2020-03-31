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

// Note: This part of the code is port from https://github.com/pingcap/schrodinger-test/tree/master/transaction/bank .
//  And it changes schrodinger-test to kubernetes-style tests.

package main

import (
	"context"
	"flag"
	"time"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/operation"
	"github.com/pingcap/tipocket/tests/bank"
)

var (
	// case config
	retryLimit  = flag.Int("retry-limit", 2, "retry count")
	accounts    = flag.Int("accounts", 1000000, "the number of accounts")
	interval    = flag.Duration("interval", 2*time.Second, "the interval")
	pessimistic = flag.Bool("pessimistic", false, "use pessimistic transaction")
	concurrency = flag.Int("concurrency", 200, "concurrency worker count")
	longTxn     = flag.Bool("long-txn", true, "enable long-term transactions")
	tables      = flag.Int("tables", 1, "the number of the tables")
	replicaRead = flag.String("tidb-replica-read", "leader", "tidb_replica_read mode, support values: leader / follower / leader-and-follower, default value: leader.")
)

func main() {
	flag.Parse()
	bank.ReplicaRead = *replicaRead

	cfg := control.Config{
		Mode:        control.ModeSelfScheduled,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}

	bankConfig := bank.Config{
		EnableLongTxn: *longTxn,
		Pessimistic:   *pessimistic,
		RetryLimit:    *retryLimit,
		Accounts:      *accounts,
		Tables:        *tables,
		Interval:      *interval,
		Concurrency:   *concurrency,
	}

	suit := util.Suit{
		Config:        &cfg,
		Provisioner:   cluster.NewK8sProvisioner(),
		ClientCreator: bank.ClientCreator{Cfg: &bankConfig},
		NemesisGens:   util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: operation.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace, fixture.Context.ImageVersion,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
