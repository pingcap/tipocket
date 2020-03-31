package operation

import (
	"github.com/pingcap/tipocket/pkg/test-infra/cdc"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/tiflash"
)

// NewDefaultCluster creates a new tidb cluster
func NewDefaultCluster(namespace, name, version string, config *fixture.ClusterConfig) Cluster {
	return tidb.New(namespace, name, version, config)
}

func NewCDCCluster(namespace, name, version string, conf *fixture.ClusterConfig) Cluster {
	cdcCluster := &CompositeCluster{ns: namespace}
	cdcCluster.AddCluster(&GroupClusterCluster{
		ops: []Cluster{
			tidb.New(namespace, name+"-upstream", version, conf),
			tidb.New(namespace, name+"-downstream", version, conf),
		},
	})
	cdcCluster.AddCluster(cdc.New(namespace, name))
	return cdcCluster
}

func NewBinlogCluster(namespace, name, version string, conf *fixture.ClusterConfig) Cluster {
	binlogCluster := &CompositeCluster{ns: namespace}
	binlogCluster.AddCluster(&GroupClusterCluster{
		ops: []Cluster{
			tidb.New(namespace, name+"-upstream", version, conf),
			tidb.New(namespace, name+"-downstream", version, conf),
		},
	})
	binlogCluster.AddCluster(cdc.New(namespace, name))
	return binlogCluster
}

func NewABTestCluster(namespace, name, versionA, versionB string, confA, confB *fixture.ClusterConfig) Cluster {
	return &GroupClusterCluster{
		ops: []Cluster{
			tidb.New(namespace, name+"-a", versionA, confA),
			tidb.New(namespace, name+"-b", versionB, confB),
		},
	}
}

func NewTiFlashCluster(namespace, name, tidbVersion string, conf *fixture.ClusterConfig) Cluster {
	c := &CompositeCluster{ns: namespace}
	c.AddCluster(tidb.New(namespace, name+"-upstream", tidbVersion, conf))
	c.AddCluster(tiflash.New(namespace, name, ""))

	return c
}
