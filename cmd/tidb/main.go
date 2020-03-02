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
	"time"

	_ "net/http/pprof"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/db/tidb"
	"github.com/pingcap/tipocket/pkg/check/porcupine"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
	tidbInfra "github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
)

var (
	clientCount  = flag.Int("client", 5, "client count")
	requestCount = flag.Int("request-count", 1000, "client test request count")
	round        = flag.Int("round", 3, "client test request round")
	runTime      = flag.Duration("run-time", 100*time.Minute, "client test run time")
	clientCase   = flag.String("case", "bank", "client test case, like bank,multi_bank")
	historyFile  = flag.String("history", "./history.log", "history file")
	qosFile      = flag.String("qos-file", "./qos.log", "qos file")
	nemesises    = flag.String("nemesis", "", "nemesis, separated by name, like random_kill,all_kill")
	mode         = flag.Int("mode", 0, "control mode, 0: mixed, 1: sequential mode, 2: self scheduled mode")
	checkerNames = flag.String("checker", "porcupine", "checker name, eg, porcupine, tidb_bank_tso")
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

	var (
		creator core.ClientCreator
		parser  = tidb.BankParser()
		model   = tidb.BankModel()
		checker core.Checker
		cfg     = control.Config{
			DB:           "noop",
			Mode:         control.Mode(*mode),
			ClientCount:  *clientCount,
			RequestCount: *requestCount,
			RunRound:     *round,
			RunTime:      *runTime,
			History:      *historyFile,
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
	switch *checkerNames {
	case "porcupine":
		checker = porcupine.Checker{}
	case "bankQoS":
		checker = tidb.BankQoSChecker(*qosFile)
	case "tidb_bank_tso":
		checker = tidb.BankTsoChecker()
	case "long_fork_checker":
		checker = tidb.LongForkChecker()
		parser = tidb.LongForkParser()
		model = nil
	//case "sequential_checker":
	//	checker = tidb.NewSequentialChecker()
	//	parser = tidb.NewSequentialParser()
	//	model = nil
	default:
		log.Fatalf("invalid checker %s", *checkerNames)
	}

	verifySuit := verify.Suit{
		Model:   model,
		Checker: checker,
		Parser:  parser,
	}
	provisioner, err := cluster.NewK8sProvisioner()
	if err != nil {
		log.Fatal(err)
	}
	suit := util.Suit{
		Config:           &cfg,
		Provisioner:      provisioner,
		ClientCreator:    creator,
		NemesisGens:      util.ParseNemesisGenerators(*nemesises),
		ClientRequestGen: util.BuildClientLoopThrottle(5 * time.Second),
		VerifySuit:       verifySuit,
		ClusterDefs:      tidbInfra.RecommendedTiDBCluster(*namespace, *namespace),
	}
	suit.Run(context.Background())
}
