package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/tests/ondup"
)

var (
	pprofAddr    = flag.String("pprof", "0.0.0.0:8080", "Pprof address")
	namespace    = flag.String("namespace", "tidb-cluster", "test namespace")
	nemesises    = flag.String("nemesis", "", "nemesis, separated by name, like random_kill,all_kill")
	hub          = flag.String("hub", "", "hub address, default to docker hub")
	imageVersion = flag.String("image-version", "latest", "image version")
	storageClass = flag.String("storage-class", "local-storage", "storage class name")
	runTime      = flag.Duration("run-time", 2*time.Minute, "client test run time")
	round        = flag.Int("round", 1, "client test request round")

	// case config
	dbName     = flag.String("db", "test", "database name")
	numRows    = flag.Int("num-rows", 10000, "number of rows")
	retryLimit = flag.Int("retry-limit", 2, "retry count")
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
		RunRound:    *round,
	}

	provisioner, err := cluster.NewK8sProvisioner()
	if err != nil {
		log.Fatal(err)
	}

	suit := util.Suit{
		Config:      &cfg,
		Provisioner: provisioner,
		ClientCreator: ondup.CaseCreator{Cfg: &ondup.Config{
			DBName:     *dbName,
			NumRows:    *numRows,
			RetryLimit: *retryLimit,
		}},
		NemesisGens: util.ParseNemesisGenerators(*nemesises),
		VerifySuit:  verify.Suit{},
		ClusterDefs: tidb.RecommendedTiDBCluster(*namespace, *namespace),
	}
	suit.Run(context.Background())
}
