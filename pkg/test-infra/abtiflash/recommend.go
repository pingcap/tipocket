package abtiflash

import (
	"fmt"

	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/tiflash"
)

// Recommendation defines 2 clusters for TiFlash abtest
type Recommendation struct {
	Cluster1 *tiflash.Recommendation
	Cluster2 *tidb.Recommendation
	NS       string
	Name     string
}

// RecommendedCluster gives a recommend cluster
func RecommendedCluster(ns, name, versionA, versionB string) *Recommendation {
	return &Recommendation{
		Cluster1: tiflash.RecommendedTiFlashCluster(ns, fmt.Sprintf("%s-a", name), versionA),
		Cluster2: tidb.RecommendedTiDBCluster(ns, fmt.Sprintf("%s-b", name), versionB),
		NS:       ns,
		Name:     name,
	}
}
