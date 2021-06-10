package main

import (
	"context"
	"flag"
	"fmt"

	test_infra "github.com/pingcap/tipocket/pkg/test-infra"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	crossregion "github.com/pingcap/tipocket/testcase/cross-region"
)

var (
	round          = flag.Int("test-round", 3, "the number of rounds to test")
	tsoRequests    = flag.Int("tso-request-count", 2000, "tso requests count for each allocator")
	pdConfTemplate = `
enable-local-tso = true
[labels]
zone = '%v'
`
	plaintextProtocolHeader = "plaintext://"
)

func main() {
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
		ClientCreator: crossregion.ClientCreator{
			Cfg: &crossregion.Config{
				DBName:      "test",
				TSORequests: *tsoRequests,
				Round:       *round,
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: provideCrossRegionCluster(),
	}
	suit.Run(context.Background())
}

func provideCrossRegionCluster() cluster.Cluster {
	namespace := fixture.Context.Namespace

	clusterPrefix := ""
	if fixture.Context.ClusterName != "" {
		clusterPrefix = fixture.Context.ClusterName + "-"
	}
	names := []string{
		clusterPrefix + "dc-1",
		clusterPrefix + "dc-2",
		clusterPrefix + "dc-3",
	}
	crossregion.DCLocations = names

	confs := []fixture.TiDBClusterConfig{
		provideConf(2, 1, 0, nil, names[0]),
		provideConf(2, 0, 0, &fixture.ClusterRef{
			Name:      names[0],
			Namespace: namespace,
		}, names[1]),
		provideConf(2, 0, 0, &fixture.ClusterRef{
			Name:      names[0],
			Namespace: namespace,
		}, names[2]),
	}
	return test_infra.NewCrossRegionTestCluster(namespace, names, confs)
}

func provideConf(pdReplicas, kvReplicas, dbReplicas int, ref *fixture.ClusterRef, dcLocation string) fixture.TiDBClusterConfig {
	cloned := fixture.Context.TiDBClusterConfig
	cloned.PDReplicas = pdReplicas
	cloned.TiKVReplicas = kvReplicas
	cloned.TiDBReplicas = dbReplicas
	cloned.Ref = ref
	cloned.PDConfig = fmt.Sprintf("%s%s", plaintextProtocolHeader, fmt.Sprintf(pdConfTemplate, dcLocation))
	return cloned
}
