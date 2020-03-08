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
	"net/http"
	"time"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/tests/bank"
)

var defaultPushMetricsInterval = 15 * time.Second

var (
	pprofAddr    = flag.String("pprof", "0.0.0.0:8080", "Pprof address")
	namespace    = flag.String("namespace", "tidb-cluster", "test namespace")
	nemesises    = flag.String("nemesis", "", "nemesis, separated by name, like random_kill,all_kill")
	hub          = flag.String("hub", "", "hub address, default to docker hub")
	imageVersion = flag.String("image-version", "latest", "image version")
	storageClass = flag.String("storage-class", "local-storage", "storage class name")
	runTime      = flag.Duration("run-time", 100*time.Minute, "client test run time")
	round        = flag.Int("round", 1, "client test request round")

	// case config
	retryLimit  = flag.Int("retry-limit", 2, "retry count")
	accounts    = flag.Int("accounts", 1000000, "the number of accounts")
	interval    = flag.Duration("interval", 2*time.Second, "the interval")
	pessimistic = flag.Bool("pessimistic", false, "use pessimistic transaction")
	concurrency = flag.Int("concurrency", 200, "concurrency worker count")
	longTxn     = flag.Bool("long-txn", true, "enable long-term transactions")
	tables      = flag.Int("tables", 1, "the number of the tables")
	// TODO: I'm not clear that this field is.
	mode = flag.String("mode", "online", "case mode, support values: online / dev, default value: online")
)

func initE2eContext() {
	fixture.E2eContext.LocalVolumeStorageClass = *storageClass
	fixture.E2eContext.HubAddress = *hub
	fixture.E2eContext.DockerRepository = "pingcap"
	fixture.E2eContext.ImageVersion = *imageVersion
}

func main() {
	flag.Parse()
	initE2eContext()
	go func() {
		http.ListenAndServe(*pprofAddr, nil)
	}()

	bankConfig := bank.Config{
		EnableLongTxn: *longTxn,
		Pessimistic:   *pessimistic,
		RetryLimit:    *retryLimit,
		Accounts:      *accounts,
		Tables:        *tables,
		Interval:      *interval,
		Concurrency:   *concurrency,
	}

	cfg := control.Config{
		Mode:        control.ModeSelfScheduled,
		ClientCount: 1,
		DB:          "noop",
	}

	provisioner, err := cluster.NewK8sProvisioner()
	if err != nil {
		log.Fatal(err)
	}

	suit := util.Suit{
		Config:        &cfg,
		Provisioner:   provisioner,
		ClientCreator: bank.CaseCreator{Cfg: &bankConfig},
		NemesisGens:   util.ParseNemesisGenerators(*nemesises),
		VerifySuit:    verify.Suit{},
		ClusterDefs:   tidb.RecommendedTiDBCluster(*namespace, *namespace),
	}
	suit.Run(context.Background())
}
