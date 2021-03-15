package testinfra

import (
	"golang.org/x/sync/errgroup"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/test-infra/binlog"
	"github.com/pingcap/tipocket/pkg/test-infra/cdc"
	"github.com/pingcap/tipocket/pkg/test-infra/dm"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/mysql"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
	"github.com/pingcap/tipocket/pkg/tidb-operator/apis/pingcap/v1alpha1"
	"github.com/pingcap/tipocket/pkg/tidb-operator/util/config"
)

// groupCluster creates clusters concurrently
type groupCluster struct {
	ops []cluster.Cluster
}

// NewGroupCluster creates a groupCluster
func NewGroupCluster(clusters ...cluster.Cluster) *groupCluster {
	return &groupCluster{ops: clusters}
}

// Apply creates the cluster
func (c *groupCluster) Apply() error {
	var g errgroup.Group
	num := len(c.ops)
	for i := 0; i < num; i++ {
		op := c.ops[i]
		g.Go(func() error {
			return op.Apply()
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

// Delete the cluster
func (c *groupCluster) Delete() error {
	var g errgroup.Group
	num := len(c.ops)
	for i := 0; i < num; i++ {
		op := c.ops[i]
		g.Go(func() error {
			return op.Delete()
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

// GetNodes returns the cluster nodes
func (c *groupCluster) GetNodes() ([]cluster.Node, error) {
	var totalNodes []cluster.Node
	for _, op := range c.ops {
		nodes, err := op.GetNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

// GetClientNodes returns the client nodes
func (c *groupCluster) GetClientNodes() ([]cluster.ClientNode, error) {
	var totalNodes []cluster.ClientNode
	for _, op := range c.ops {
		nodes, err := op.GetClientNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

// compositeCluster creates clusters sequentially
type compositeCluster struct {
	ops []cluster.Cluster
}

// NewCompositeCluster creates a compositeCluster
func NewCompositeCluster(clusters ...cluster.Cluster) *compositeCluster {
	return &compositeCluster{ops: clusters}
}

// Apply creates the cluster
func (c *compositeCluster) Apply() error {
	for _, op := range c.ops {
		if err := op.Apply(); err != nil {
			return err
		}
	}
	return nil
}

// Delete the cluster
func (c *compositeCluster) Delete() error {
	for _, op := range c.ops {
		if err := op.Delete(); err != nil {
			return err
		}
	}
	return nil
}

// GetNodes returns the cluster nodes
func (c *compositeCluster) GetNodes() ([]cluster.Node, error) {
	var totalNodes []cluster.Node
	for _, op := range c.ops {
		nodes, err := op.GetNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

// GetClientNodes returns the client nodes
func (c *compositeCluster) GetClientNodes() ([]cluster.ClientNode, error) {
	var totalNodes []cluster.ClientNode
	for _, op := range c.ops {
		nodes, err := op.GetClientNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

// NewDefaultCluster creates a new TiDB cluster
func NewDefaultCluster(namespace, name string, config fixture.TiDBClusterConfig) cluster.Cluster {
	return tidb.New(namespace, name, config)
}

// NewCDCCluster creates two TiDB clusters with CDC
func NewCDCCluster(namespace, name string, upstreamConfig, downstreamConfig fixture.TiDBClusterConfig) cluster.Cluster {
	return NewCompositeCluster(
		NewGroupCluster(
			tidb.New(namespace, name+"-upstream", upstreamConfig),
			tidb.New(namespace, name+"-downstream", downstreamConfig),
		),
		cdc.New(namespace, name, name+"-upstream", name+"-downstream"),
	)
}

// NewBinlogCluster creates two TiDB clusters with Binlog
func NewBinlogCluster(namespace, name string, conf fixture.TiDBClusterConfig) cluster.Cluster {
	up := tidb.New(namespace, name+"-upstream", conf)
	upstream := up.GetTiDBCluster()
	upstream.Spec.TiDB.BinlogEnabled = pointer.BoolPtr(true)
	upstream.Spec.Pump = &v1alpha1.PumpSpec{
		Replicas:             3,
		ResourceRequirements: fixture.WithStorage(fixture.Small, "10Gi"),
		StorageClassName:     &fixture.Context.LocalVolumeStorageClass,
		ComponentSpec: v1alpha1.ComponentSpec{
			Image: util.BuildImage("tidb-binlog", fixture.Context.TiDBClusterConfig.ImageVersion, fixture.Context.BinlogConfig.Image),
		},
		Config: &config.GenericConfig{},
	}

	return NewCompositeCluster(
		NewGroupCluster(up, tidb.New(namespace, name+"-downstream", conf)),
		binlog.New(namespace, name),
	)
}

// NewDMCluster creates two upstream MySQL instances and a downstream TiDB cluster with DM.
func NewDMCluster(namespace, name string, dmConf fixture.DMConfig, tidbConf fixture.TiDBClusterConfig) cluster.Cluster {
	up1 := mysql.New(namespace, name+"-mysql1", dmConf.MySQLConf)
	up2 := mysql.New(namespace, name+"-mysql2", dmConf.MySQLConf)
	down := tidb.New(namespace, name+"-tidb", tidbConf)

	return NewCompositeCluster(
		up1, up2, down,
		dm.New(namespace, name+"-dm", dmConf))
}

// NewABTestCluster creates two TiDB clusters to do AB Test
func NewABTestCluster(namespace, name string, confA, confB fixture.TiDBClusterConfig) cluster.Cluster {
	return NewGroupCluster(
		tidb.New(namespace, name+"-a", confA),
		tidb.New(namespace, name+"-b", confB),
	)
}

// NewTiFlashCluster creates a TiDB cluster with TiFlash
func NewTiFlashCluster(namespace, name string, conf fixture.TiDBClusterConfig) cluster.Cluster {
	return tidb.New(namespace, name, conf)
}

// NewTiFlashABTestCluster creates two TiDB clusters to do AB Test, one with TiFlash
func NewTiFlashABTestCluster(namespace, name string, confA, confB fixture.TiDBClusterConfig) cluster.Cluster {
	return NewGroupCluster(
		NewTiFlashCluster(namespace, name+"-a", confA),
		tidb.New(namespace, name+"-b", confB),
	)
}

// NewTiFlashCDCABTestCluster creates two TiDB clusters to do AB Test, one with TiFlash
// This also includes a CDC cluster between the two TiDB clusters.
func NewTiFlashCDCABTestCluster(namespace, name string, confA, confB fixture.TiDBClusterConfig) cluster.Cluster {
	return NewCompositeCluster(
		NewGroupCluster(
			NewTiFlashCluster(namespace, name+"-upstream", confA),
			tidb.New(namespace, name+"-downstream", confB),
		),
		cdc.New(namespace, name, name+"-upstream", name+"-downstream"),
	)
}

// NewCrossRegionTestCluster creates multi tidbcluster in different regions
func NewCrossRegionTestCluster(namespace string, names []string, confs []fixture.TiDBClusterConfig) cluster.Cluster {
	var clusters []cluster.Cluster
	for i, conf := range confs {
		clusters = append(clusters, tidb.New(namespace, names[i], conf))
	}
	return NewGroupCluster(clusters...)
}
