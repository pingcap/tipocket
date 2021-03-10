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

	logs "github.com/pingcap/tipocket/logsearch/pkg/logs"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/verify"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	client "github.com/pingcap/tipocket/testcase/bank-two"
)

var (
	accounts    = flag.Int("accounts", 1000000, "the number of accounts")
	interval    = flag.Duration("interval", 2*time.Second, "the interval")
	concurrency = flag.Int("concurrency", 200, "concurrency worker count")
	retryLimit  = flag.Int("retry-limit", 200, "retry count")
	longTxn     = flag.Bool("long-txn", true, "enable long-term transactions")
	contention  = flag.String("contention", "low", "contention level, support values: high / low, default value: low")
	pessimistic = flag.Bool("pessimistic", false, "use pessimistic transaction")
	minLength   = flag.Int("min-value-length", 0, "minimum value inserted into rocksdb")
	maxLength   = flag.Int("max-value-length", 128, "maximum value inserted into rocksdb")
	replicaRead = flag.String("tidb-replica-read", "leader", "tidb_replica_read mode, support values: leader / follower / leader-and-follower, default value: leader.")
	dbname      = flag.String("dbname", "test", "name of database to test")
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
		ClientCreator: client.ClientCreator{
			Cfg: &client.Config{
				NumAccounts:   *accounts,
				Interval:      *interval,
				Concurrency:   *concurrency,
				RetryLimit:    *retryLimit,
				MinLength:     *minLength,
				MaxLength:     *maxLength,
				EnableLongTxn: *longTxn,
				Contention:    *contention,
				Pessimistic:   *pessimistic,
				ReplicaRead:   *replicaRead,
				DbName:        *dbname,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		VerifySuit:  verify.Suit{},
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
		LogsClient: logs.NewDiagnosticLogClient(),
	}
	suit.Run(context.Background())
}
