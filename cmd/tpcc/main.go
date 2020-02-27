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
	"math"
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/pingcap/tipocket/pkg/core"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/db/tidb"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
	tidbInfra "github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
)

var (
	clientCount  = flag.Int("client", 5, "client count")
	requestCount = flag.Int("request-count", math.MaxInt64, "client test request count")
	round        = flag.Int("round", 3, "client test request round")
	runTime      = flag.Duration("run-time", 10*time.Minute, "client test run time")
	historyFile  = flag.String("history", "./history.log", "history file")
	qosFile      = flag.String("qosFile", "./qos.log", "qos file")
	nemesises    = flag.String("nemesis", "", "nemesis, separated by name, like random_kill,all_kill")
	checkerName  = flag.String("checker", "consistency", "consistency or qos")
	pprofAddr    = flag.String("pprof", "0.0.0.0:8080", "Pprof address")
	namespace    = flag.String("namespace", "tidb-cluster", "test namespace")
	hub          = flag.String("hub", "", "hub address, default to docker hub")
	imageVersion = flag.String("image-version", "latest", "image version")
	storageClass = flag.String("storage-class", "local-storage", "storage class name")
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
		DB:           "noop",
		Mode:         control.ModeSequential,
		ClientCount:  *clientCount,
		RequestCount: *requestCount,
		RunRound:     *round,
		RunTime:      *runTime,
		History:      *historyFile,
	}
	clientCreator := &tidb.TPCCClientCreator{}
	creator := clientCreator
	var checker core.Checker
	switch *checkerName {
	case "consistency":
		checker = &tidb.TPCCChecker{CreatorRef: clientCreator}
	case "qos":
		checker = tidb.TPCCQosChecker(time.Minute, *qosFile)
	}
	verifySuit := verify.Suit{
		Model:   nil,
		Checker: checker,
		Parser:  tidb.TPCCParser(),
	}
	provisioner, err := cluster.NewK8sProvisioner()
	if err != nil {
		log.Fatal(err)
	}

	var waitWarmUpNemesisGens []core.NemesisGenerator
	for _, gen := range util.ParseNemesisGenerators(*nemesises) {
		waitWarmUpNemesisGens = append(waitWarmUpNemesisGens, core.DelayNemesisGenerator{
			Gen:   gen,
			Delay: time.Minute * time.Duration(2),
		})
	}
	suit := util.Suit{
		Config:        &cfg,
		Provisioner:   provisioner,
		ClientCreator: creator,
		NemesisGens:   waitWarmUpNemesisGens,
		VerifySuit:    verifySuit,
		ClusterDefs:   tidbInfra.RecommendedTiDBCluster(*namespace, *namespace),
	}
	suit.Run(context.Background())
}
