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
	"strconv"
	"strings"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/tests/pessimistic"
	"github.com/pingcap/tipocket/tests/pessimistic/hongbao"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

const caseName = "txn-rand-pessimistic"

var (
	pprofAddr    = flag.String("pprof", "0.0.0.0:8080", "Pprof address")
	namespace    = flag.String("namespace", "tidb-cluster", "test namespace")
	nemesises    = flag.String("nemesis", "", "nemesis, separated by name, like random_kill,all_kill")
	hub          = flag.String("hub", "", "hub address, default to docker hub")
	imageVersion = flag.String("image-version", "latest", "image version")
	storageClass = flag.String("storage-class", "local-storage", "storage class name")

	// case config
	randTxnDBName      = flag.String("rand-txn-db-name", "randtxn", "database name for random transaction")
	randTxnConcurrency = flag.Int("rand-txn-concurrency", 32, "concurrency for random transaction")
	tableNum           = flag.Int("table-num", 1, "table number")
	tableSize          = flag.Uint64("table-size", 400000, "table size")
	operationCount     = flag.Uint64("operation-count", 40000, "count of test operations")
	prepareStmt        = flag.Bool("prepare-stmt", true, "use prepared statement")
	ignoreO            = flag.String("ignore-o", "9007,1105", "ignored error code for optimistic transaction, separated by comma")
	ignoreP            = flag.String("ignore-p", "1213", "ignored error code for pessimistic transaction, separated by comma")
	insertDelete       = flag.Bool("insert-delete", false, "run insert delete transactions")

	hongbaoDBName      = flag.String("hongbao-db-name", "hongbao", "database name for hongbao case")
	userNum            = flag.Int("user-num", 1000, "number of users in total")
	friendNum          = flag.Int("friend-num", 20, "number of friends for each user")
	groupNum           = flag.Int("group-num", 20, "number of groups in total")
	groupMemberNum     = flag.Int("group-member-num", 20, "number of members for each groups")
	hongbaoConcurrency = flag.Int("hongbao-concurrency", 32, "concurrency for hongbao case")
	hongbaoNum         = flag.Int("hongbao-num", 5, "number of hongbao for each concurrency")

	txnMode  = flag.String("txn-mode", "mix", "transaction mode, mix|pessimistic|optimistic")
	caseMode = flag.String("case-mode", "online", "case mode, support values: online / dev, default value: online")
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
	}

	ignoreCodesO, err := splitToSlice(*ignoreO)
	if err != nil {
		log.Fatalf("[%s] parse argment error: %v", caseName, err)
	}
	ignoreCodesP, err := splitToSlice(*ignoreP)
	if err != nil {
		log.Fatalf("[%s] parse argment error: %v", caseName, err)
	}

	provisioner, err := cluster.NewK8sProvisioner()
	if err != nil {
		log.Fatal(err)
	}
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: provisioner,
		ClientCreator: pessimistic.CaseCreator{Cfg: &pessimistic.Config{
			TxnMode: *txnMode,
			PessimisticCaseConfig: pessimistic.PessimisticCaseConfig{
				DBName:         *randTxnDBName,
				Concurrency:    *randTxnConcurrency,
				TableNum:       *tableNum,
				TableSize:      *tableSize,
				OperationCount: *operationCount,
				Mode:           *txnMode,
				InsertDelete:   *insertDelete,
				IgnoreCodesO:   ignoreCodesO,
				IgnoreCodesP:   ignoreCodesP,
				UsePrepareStmt: *prepareStmt,
			},
			HongbaoCaseConfig: hongbao.HongbaoCaseConfig{
				DBName:         *hongbaoDBName,
				Concurrency:    *hongbaoConcurrency,
				UserNum:        *userNum,
				FriendNum:      *friendNum,
				GroupNum:       *groupNum,
				GroupMemberNum: *groupMemberNum,
				HongbaoNum:     *hongbaoNum,
				IgnoreCodesO:   ignoreCodesO,
				IgnoreCodesP:   ignoreCodesP,
				TxnMode:        *txnMode,
			},
		}},
		NemesisGens: util.ParseNemesisGenerators(*nemesises),
		VerifySuit:  verify.Suit{},
		ClusterDefs: tidb.RecommendedTiDBCluster(*namespace, *namespace),
	}
	suit.Run(context.Background())
}

func splitToSlice(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	ret := make([]int, len(parts))
	for i, part := range parts {
		iv, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		ret[i] = iv
	}
	return ret, nil
}
