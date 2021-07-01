package main

import (
	"context"
	"flag"
	"log"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/check/porcupine"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/verify"

	rawkv_linearizability "github.com/pingcap/tipocket/testcase/rawkv-linearizability"
)

var (
	keyStart        = flag.Int("KeyStart", 0, "the start of the key")
	keyNum          = flag.Int("KeyNum", 100000, "the key range number")
	readProbability = flag.Int("ReadProbability", 60, "the probaility of read request")
	writeProbaility = flag.Int("WriteProbaility", 35, "the probaility of write request, the remaining part is the probaility of delete request")
	valueNum10KB    = flag.Int("ValueNum10KB", 400, "10KB value kind number")
	valueNum100KB   = flag.Int("ValueNum100KB", 400, "100KB value kind number")
	valueNum1MB     = flag.Int("ValueNum1MB", 200, "1MB value kind number")
	valueNum5MB     = flag.Int("ValueNum5MB", 40, "5MB value kind number")
)

func main() {
	flag.Parse()

	var checkers []core.Checker
	checkers = append(checkers, porcupine.Checker{})
	// TODO should add more checker

	verifySuit := verify.Suit{
		Model:   rawkv_linearizability.RawkvModel(),
		Checker: core.MultiChecker("rawkv-linearizability checkers", checkers...),
		Parser:  rawkv_linearizability.RawkvParser(),
	}
	cfg := control.Config{
		Mode:         control.Mode(control.ModeOnSchedule),
		ClientCount:  fixture.Context.ClientCount,
		RequestCount: fixture.Context.RequestCount,
		RunRound:     fixture.Context.RunRound,
		RunTime:      fixture.Context.RunTime,
		History:      fixture.Context.HistoryFile,
	}
	log.Printf("request count:%v", cfg.RequestCount)
	randomValues := rawkv_linearizability.GenerateRandomValueString(rawkv_linearizability.RandomValueConfig{
		ValueNum10KB:  *valueNum10KB,
		ValueNum100KB: *valueNum100KB,
		ValueNum1MB:   *valueNum1MB,
		ValueNum5MB:   *valueNum5MB,
	})
	//kvs := []string{"127.0.0.1:20160", "127.0.0.1:20162", "127.0.0.1:20161"}
	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: rawkv_linearizability.RawkvClientCreator{
			Cfg: rawkv_linearizability.Config{
				KeyStart:        *keyStart,
				KeyNum:          *keyNum,
				ReadProbability: *readProbability,
				WriteProbaility: *writeProbaility,
			},
			RandomValues: &randomValues,
		},
		NemesisGens:      util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClientRequestGen: util.OnClientLoop,
		VerifySuit:       verifySuit,
		ClusterDefs:      test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.ClusterName, fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
