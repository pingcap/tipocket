package main

import (
	"context"
	"flag"
	"fmt"
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
	concurrency   = flag.Int("concurrency", 200, "concurrent worker count")
	dataPerWorker = flag.Int("data-per-worker", 2000, "data count for each concurrent worker")
	ttlCandidates arrayIntFlags

	// Some internal magic numbers
	toleranceOfTTL     = flag.Uint64("tolerance-of-ttl", 60, "tolerance when comparing TTL with expected TTL.")
	zeroTTLVerifyDelay = flag.Uint64("zero-ttl-verify-delay", 600, "the timeout to verify the input when ttl is set to zero")
)

func (a *arrayIntFlags) String() string {
	str := ""
	for i := range *a {
		str += ", " + fmt.Sprint(i)
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

// Usage: ./bin/ttl --concurrency 200 --data-per-worker 1000 --ttl 0 --ttl 600 --ttl 3600
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
				Concurrency:        *concurrency,
				DataPerWorker:      *dataPerWorker,
				TTLCandidates:      ttlCandidates,
				ToleranceOfTTL:     *toleranceOfTTL,
				ZeroTTLVerifyDelay: *zeroTTLVerifyDelay,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
