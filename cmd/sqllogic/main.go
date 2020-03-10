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
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/pingcap/tipocket/tests/sqllogictest"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
)

var (
	pprofAddr    = flag.String("pprof", "0.0.0.0:8080", "Pprof address")
	namespace    = flag.String("namespace", "tidb-cluster", "test namespace")
	nemesises    = flag.String("nemesis", "", "nemesis, separated by name, like random_kill,all_kill")
	hub          = flag.String("hub", "", "hub address, default to docker hub")
	imageVersion = flag.String("image-version", "latest", "image version")
	storageClass = flag.String("storage-class", "local-storage", "storage class name")
	runTime      = flag.Duration("run-time", 100*time.Minute, "client test run time")

	// case config
	sqllogicCaseURL = flag.String("p", "", "case url")
	taskCount       = flag.Int("t", 2, "concurrency")
	skipError       = flag.Bool("skip-error", false, "skip error for query test")
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

	cfg := control.Config{
		Mode:        control.ModeSelfScheduled,
		ClientCount: 1,
		DB:          "noop",
		RunTime:     *runTime,
		RunRound:    1,
	}

	provisioner, err := cluster.NewK8sProvisioner()
	if err != nil {
		log.Fatal(err)
	}
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: provisioner,
		ClientCreator: &sqllogictest.CaseCreator{
			Config: &sqllogictest.Config{
				SkipError: *skipError,
				TaskCount: 10,
				CaseURL:   *sqllogicCaseURL,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(*nemesises),
		VerifySuit:  verify.Suit{},
		ClusterDefs: tidb.RecommendedTiDBCluster(*namespace, *namespace),
	}
	suit.Run(context.Background())
}
