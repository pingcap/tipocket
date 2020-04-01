package testinfra

import (
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util/config"
	"golang.org/x/sync/errgroup"
	"k8s.io/utils/pointer"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/binlog"
	"github.com/pingcap/tipocket/pkg/test-infra/cdc"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/tiflash"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
)

// GroupClusterCluster ...
type GroupClusterCluster struct {
	ops []clusterTypes.Cluster
	ns  string
}

// Namespace ...
func (c GroupClusterCluster) Namespace() string {
	return c.ns
}

// Apply ...
func (c *GroupClusterCluster) Apply() error {
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

// Delete ...
func (c *GroupClusterCluster) Delete() error {
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

// GetNodes ...
func (c *GroupClusterCluster) GetNodes() ([]clusterTypes.Node, error) {
	var totalNodes []clusterTypes.Node
	for _, op := range c.ops {
		nodes, err := op.GetNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

// GetClientNodes ...
func (c *GroupClusterCluster) GetClientNodes() ([]clusterTypes.ClientNode, error) {
	var totalNodes []clusterTypes.ClientNode
	for _, op := range c.ops {
		nodes, err := op.GetClientNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

// WithCluster ...
func (c *GroupClusterCluster) WithCluster(cluster clusterTypes.Cluster) {
	c.ops = append(c.ops, cluster)
}

// CompositeCluster ...
type CompositeCluster struct {
	ops []clusterTypes.Cluster
	ns  string
}

func (c CompositeCluster) Namespace() string {
	return c.ns
}

func (c *CompositeCluster) Apply() error {
	for _, op := range c.ops {
		if err := op.Apply(); err != nil {
			return err
		}
	}
	return nil
}

func (c *CompositeCluster) Delete() error {
	for _, op := range c.ops {
		if err := op.Delete(); err != nil {
			return err
		}
	}
	return nil
}

func (c *CompositeCluster) GetNodes() ([]clusterTypes.Node, error) {
	var totalNodes []clusterTypes.Node
	for _, op := range c.ops {
		nodes, err := op.GetNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

func (c *CompositeCluster) GetClientNodes() ([]clusterTypes.ClientNode, error) {
	var totalNodes []clusterTypes.ClientNode
	for _, op := range c.ops {
		nodes, err := op.GetClientNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

func (c *CompositeCluster) AddCluster(cluster clusterTypes.Cluster) {
	c.ops = append(c.ops, cluster)
}

// NewDefaultCluster creates a new tidb cluster
func NewDefaultCluster(namespace, name string, config fixture.TiDBClusterConfig) clusterTypes.Cluster {
	return tidb.New(namespace, name, config)
}

func NewCDCCluster(namespace, name string, conf fixture.TiDBClusterConfig) clusterTypes.Cluster {
	cdcCluster := &CompositeCluster{ns: namespace}
	cdcCluster.AddCluster(&GroupClusterCluster{
		ops: []clusterTypes.Cluster{
			tidb.New(namespace, name+"-upstream", conf),
			tidb.New(namespace, name+"-downstream", conf),
		},
	})
	cdcCluster.AddCluster(cdc.New(namespace, name))
	return cdcCluster
}

func NewBinlogCluster(namespace, name string, conf fixture.TiDBClusterConfig) clusterTypes.Cluster {
	binlogCluster := &CompositeCluster{ns: namespace}
	up := tidb.New(namespace, name+"-upstream", conf)
	upstream := up.GetTiDBCluster()
	binlogCluster.AddCluster(&GroupClusterCluster{
		ops: []clusterTypes.Cluster{up, tidb.New(namespace, name+"-downstream", conf)},
	})
	upstream.Spec.TiDB.BinlogEnabled = pointer.BoolPtr(true)
	upstream.Spec.Pump = &v1alpha1.PumpSpec{
		Replicas:             3,
		ResourceRequirements: fixture.WithStorage(fixture.Small, "10Gi"),
		StorageClassName:     &fixture.Context.LocalVolumeStorageClass,
		ComponentSpec: v1alpha1.ComponentSpec{
			Image: util.BuildBinlogImage("tidb-binlog"),
		},
		GenericConfig: config.GenericConfig{
			Config: map[string]interface{}{},
		},
	}
	binlogCluster.AddCluster(binlog.New(namespace, name))
	return binlogCluster
}

func NewABTestCluster(namespace, name string, confA, confB fixture.TiDBClusterConfig) clusterTypes.Cluster {
	return &GroupClusterCluster{
		ops: []clusterTypes.Cluster{
			tidb.New(namespace, name+"-a", confA),
			tidb.New(namespace, name+"-b", confB),
		},
		ns: namespace,
	}
}

func NewTiFlashCluster(namespace, name string, conf fixture.TiDBClusterConfig) clusterTypes.Cluster {
	c := &CompositeCluster{ns: namespace}
	t := tidb.New(namespace, name, conf)
	tc := t.GetTiDBCluster()
	// To make TiFlash work, we need to enable placement rules in pd.
	tc.Spec.PD.Config = &v1alpha1.PDConfig{
		Replication: &v1alpha1.PDReplicationConfig{
			EnablePlacementRules: pointer.BoolPtr(true),
		},
	}
	c.AddCluster(t)
	c.AddCluster(tiflash.New(namespace, name))

	return c
}

func NewTiFlashABTestCluster(namespace, name string, confA, confB fixture.TiDBClusterConfig) clusterTypes.Cluster {
	c := &CompositeCluster{ns: namespace}
	c.AddCluster(&GroupClusterCluster{ops: []clusterTypes.Cluster{
		NewTiFlashCluster(namespace, name+"-a", confA),
		tidb.New(namespace, name+"-b", confB),
	}, ns: namespace})

	return c
}

func NewTiFlashCDCABTestCluster(namespace, name string, confA, confB fixture.TiDBClusterConfig) clusterTypes.Cluster {
	c := &CompositeCluster{ns: namespace}
	c.AddCluster(&GroupClusterCluster{ops: []clusterTypes.Cluster{
		NewTiFlashCluster(namespace, name+"-upstream", confA),
		tidb.New(namespace, name+"-downstream", confB),
	}, ns: namespace})
	c.AddCluster(cdc.New(namespace, name))
	return c
}
