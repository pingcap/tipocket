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
	rawkvlinearizability "github.com/pingcap/tipocket/tests/rawkv-linearizability"
)

var (
	KeyStart        = flag.Int("KeyStart", 0, "the start of the key")
	KeyNum          = flag.Int("KeyNum", 100000, "the key range number")
	ReadProbability = flag.Int("ReadProbability", 60, "the probaility of read request")
	WriteProbaility = flag.Int("WriteProbaility", 35, "the probaility of write request, the remaining part is the probaility of delete request")
	ValueNum10KB    = flag.Int("ValueNum10KB", 400, "10KB value kind number")
	ValueNum100KB   = flag.Int("ValueNum100KB", 400, "100KB value kind number")
	ValueNum1MB     = flag.Int("ValueNum1MB", 200, "1MB value kind number")
	ValueNum5MB     = flag.Int("ValueNum5MB", 40, "5MB value kind number")
)

func main() {
	flag.Parse()

	var checkers []core.Checker
	checkers = append(checkers, porcupine.Checker{})
	// TODO should add more checker

	verifySuit := verify.Suit{
		Model:   rawkvlinearizability.RawkvModel(),
		Checker: core.MultiChecker("rawkv-linearizability checkers", checkers...),
		Parser:  rawkvlinearizability.RawkvParser(),
	}
	cfg := control.Config{
		Mode:         control.Mode(control.ModeMixed),
		ClientCount:  fixture.Context.ClientCount,
		RequestCount: fixture.Context.RequestCount,
		RunRound:     fixture.Context.RunRound,
		RunTime:      fixture.Context.RunTime,
		History:      fixture.Context.HistoryFile,
	}
	log.Printf("request count:%v", cfg.RequestCount)
	randomValues := rawkvlinearizability.GenerateRandomValueString(rawkvlinearizability.RandomValueConfig{
		ValueNum10KB:  *ValueNum10KB,
		ValueNum100KB: *ValueNum100KB,
		ValueNum1MB:   *ValueNum1MB,
		ValueNum5MB:   *ValueNum5MB,
	})
	//kvs := []string{"127.0.0.1:20160", "127.0.0.1:20162", "127.0.0.1:20161"}
	suit := util.Suit{
		Config:      &cfg,
		Provisioner: cluster.NewK8sProvisioner(),
		//Provisioner: cluster.NewLocalClusterProvisioner([]string{"127.0.0.1:4000"}, []string{"127.0.0.1:2379"}, kvs),
		ClientCreator: rawkvlinearizability.RawkvClientCreator{
			Cfg: rawkvlinearizability.Config{
				KeyStart:        *KeyStart,
				KeyNum:          *KeyNum,
				ReadProbability: *ReadProbability,
				WriteProbaility: *WriteProbaility,
			},
			RandomValues: &randomValues,
		},
		NemesisGens:      util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClientRequestGen: util.OnClientLoop,
		VerifySuit:       verifySuit,
		ClusterDefs:      test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace, fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
