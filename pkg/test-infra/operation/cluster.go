package operation

import (
	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"golang.org/x/sync/errgroup"
)

type GroupClusterCluster struct {
	ops []Cluster
	ns  string
}

func (c GroupClusterCluster) Namespace() string {
	return c.ns
}

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

func (c *GroupClusterCluster) GetNodes() ([]clusterTypes.Node, error) {
	totalNodes := []clusterTypes.Node{}
	for _, op := range c.ops {
		nodes, err := op.GetNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

func (c *GroupClusterCluster) GetClientNodes() ([]clusterTypes.ClientNode, error) {
	totalNodes := []clusterTypes.ClientNode{}
	for _, op := range c.ops {
		nodes, err := op.GetClientNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

func (c *GroupClusterCluster) WithCluster(cluster Cluster) {
	c.ops = append(c.ops, cluster)
}

type CompositeCluster struct {
	ops []Cluster
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
	totalNodes := []clusterTypes.Node{}
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
	totalNodes := []clusterTypes.ClientNode{}
	for _, op := range c.ops {
		nodes, err := op.GetClientNodes()
		if err != nil {
			return nil, err
		}
		totalNodes = append(totalNodes, nodes...)
	}
	return totalNodes, nil
}

func (c *CompositeCluster) AddCluster(cluster Cluster) {
	c.ops = append(c.ops, cluster)
}