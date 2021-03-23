package main

import (
	"context"
	"flag"
	"strconv"

	"github.com/pingcap/tipocket/cmd/util"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	testcase "github.com/pingcap/tipocket/testcase/ttl"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

type arrayIntFlags []uint64

var (
	concurrency   = flag.Int("concurrency", 200, "concurrency worker count")
	dataPerWorker = flag.Int("data-per-worker", 2000, "data count for each concurrency worker")
	ttlCandidates arrayIntFlags
)

func (a *arrayIntFlags) String() string {
	str := ""
	for i := range *a {
		str += ", " + string(i)
	}
	return str
}

func (a *arrayIntFlags) Set(value string) error {
	i, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return err
	}
	*a = append(*a, i)
	return nil
}

func main() {
	flag.Var(&ttlCandidates, "ttl", "Time to live, If pass multiple ttls, each worker would randomly pick one")
	flag.Parse()

	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}

	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: testcase.ClientCreator{
			Cfg: &testcase.Config{
				Concurrency:   *concurrency,
				DataPerWorker: *dataPerWorker,
				TTLCandidates: ttlCandidates,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
