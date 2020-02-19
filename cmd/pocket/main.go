// Copyright 2019 PingCAP, Inc.
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

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/pocket/creator"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/binlog"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
	"github.com/pingcap/tipocket/pkg/verify"
)

var (
	configPath    = flag.String("config", "", "config file path")
	pprofAddr     = flag.String("pprof", "0.0.0.0:8080", "Pprof address")
	namespace     = flag.String("namespace", "tidb-cluster", "test namespace")
	hub           = flag.String("hub", "", "hub address, default to docker hub")
	imageVersion  = flag.String("image-version", "latest", "image version")
	binlogVersion = flag.String("binlog-version", "", `overwrite "-image-version" flag for drainer`)
	storageClass  = flag.String("storage-class", "local-storage", "storage class name")
)

func initE2eContext() {
	fixture.E2eContext.LocalVolumeStorageClass = *storageClass
	fixture.E2eContext.HubAddress = *hub
	fixture.E2eContext.DockerRepository = "pingcap"
	fixture.E2eContext.ImageVersion = *imageVersion
	fixture.E2eContext.BinlogVersion = *binlogVersion
}

func main() {
	flag.Parse()
	initE2eContext()
	go func() {
		http.ListenAndServe(*pprofAddr, nil)
	}()

	cfg := control.Config{
		Mode:       control.ModeSelfScheduled,
		DB:         "noop",
		CaseConfig: *configPath,
	}

	verifySuit := verify.Suit{
		Model:   &core.NoopModel{},
		Checker: core.NoopChecker{},
		Parser:  nil,
	}
	provisioner, err := cluster.NewK8sProvisioner()
	if err != nil {
		log.Fatal(err)
	}
	suit := util.Suit{
		Config:        &cfg,
		Provisioner:   provisioner,
		ClientCreator: creator.PocketCreator{},
		VerifySuit:    verifySuit,
		Cluster:       binlog.RecommendedBinlogCluster(*namespace, *namespace),
	}
	suit.Run(context.Background())
}
