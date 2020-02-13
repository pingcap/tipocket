package binlog

import (
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"
)

// ClusterRecommendation defines binlog cluster
type ClusterRecommendation struct {
	Upstream   *tidb.TiDBClusterRecommendation
	Downstream *tidb.TiDBClusterRecommendation
}

// RecommendedBinlogCluster create cluster with binlog
func RecommendedBinlogCluster(ns, name string) *ClusterRecommendation {
	var (
		upstream   = tidb.RecommendedTiDBCluster(ns, fmt.Sprintf("%s-upstream", name))
		downstream = tidb.RecommendedTiDBCluster(ns, fmt.Sprintf("%s-downstream", name))
	)

	upstream.TidbCluster.Spec.TiDB.BinlogEnabled = true
	upstream.TidbCluster.Spec.Pump = &v1alpha1.PumpSpec{
		Replicas:         3,
		Resources:        fixture.WithStorage(fixture.Small, "10Gi"),
		StorageClassName: fixture.E2eContext.LocalVolumeStorageClass,
		ComponentSpec: v1alpha1.ComponentSpec{
			Image: buildImage("tidb-binlog"),
		},
	}

	return &ClusterRecommendation{
		Upstream:   upstream,
		Downstream: downstream,
	}
}

func buildImage(name string) string {
	var b strings.Builder
	if fixture.E2eContext.HubAddress != "" {
		fmt.Fprintf(&b, "%s/", fixture.E2eContext.HubAddress)
	}
	b.WriteString(fixture.E2eContext.DockerRepository)
	b.WriteString("/")
	b.WriteString(name)
	b.WriteString(":")
	b.WriteString(fixture.E2eContext.ImageVersion)
	return b.String()
}
