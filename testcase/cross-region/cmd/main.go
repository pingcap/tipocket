package main

import (
	"context"
	"flag"

	// use mysql
	_ "github.com/go-sql-driver/mysql"

	"github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	crossregion "github.com/pingcap/tipocket/testcase/cross-region"
	corev1 "k8s.io/api/core/v1"
)

func main() {
	flag.Parse()
	cfg := control.Config{
		Mode:        control.ModeStandard,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}
	np := corev1.ServiceTypeNodePort
	fixture.Context.TiDBClusterConfig.PDReplicas = 1
	fixture.Context.TiDBClusterConfig.TiKVReplicas = 1
	fixture.Context.TiDBClusterConfig.TiDBReplicas = 1
	fixture.Context.Namespace = "cross-region"
	fixture.Context.Name = "cross-region"
	fixture.Context.TiDBClusterConfig.PDStorageClassName = "shared-nvme-disks"
	fixture.Context.TiDBClusterConfig.TiKVStorageClassName = "nvme-disks"
	fixture.Context.TiDBClusterConfig.LogStorageClassName = "shared-sas-disks"
	fixture.Context.TiDBClusterConfig.PDSvcType = &np
	suit := util.Suit{
		Config:   &cfg,
		Provider: cluster.NewDefaultClusterProvider(),
		ClientCreator: crossregion.ClientCreator{
			Cfg: &crossregion.Config{
				DBName: "test",
			},
		},
		NemesisGens: util.ParseNemesisGenerators(fixture.Context.Nemesis),
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Name,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
