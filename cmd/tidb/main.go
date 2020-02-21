package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

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
	runTime      = flag.Duration("run-time", 10*time.Minute, "client test run time")
	clientCase   = flag.String("case", "bank", "client test case, like bank,multi_bank")
	historyFile  = flag.String("history", "./history.log", "history file")
	profFile     = flag.String("prof", "./prof.log", "service quality prof file")
	nemesises    = flag.String("nemesis", "", "nemesis, separated by name, like random_kill,all_kill")
	checkerNames = flag.String("checker", "porcupine", "checker name, eg, porcupine, tidb_bank_tso")
	pprofAddr    = flag.String("pprof", "0.0.0.0:8080", "Pprof address")
	namespace    = flag.String("namespace", "tidb-cluster", "test namespace")
	//chaosNamespace = flag.String("chaos-ns", "chaos-testing", "test chaos namespace")
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

	//if chaosNamespace == nil || *chaosNamespace == "" {
	//	chaosNamespace = namespace
	//}

	cfg := control.Config{
		DB:           "noop",
		ClientCount:  *clientCount,
		RequestCount: *requestCount,
		RunRound:     *round,
		RunTime:      *runTime,
		History:      *historyFile,
	}

	var creator core.ClientCreator
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

	parser := tidb.BankParser()
	model := tidb.BankModel()
	withProf := false
	var checker core.Checker
	switch *checkerNames {
	case "porcupine":
		checker = porcupine.Checker{}
	case "service_quality":
		checker = tidb.BankServiceQualityChecker(*profFile)
		withProf = true
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
		Config:        &cfg,
		Provisioner:   provisioner,
		ClientCreator: creator,
		Nemesises:     *nemesises,
		WithProf:      withProf,
		VerifySuit:    verifySuit,
		Cluster:       tidbInfra.RecommendedTiDBCluster(*namespace, *namespace),
	}
	suit.Run(context.Background())
}
