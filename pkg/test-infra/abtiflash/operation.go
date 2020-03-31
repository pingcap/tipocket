package abtiflash

import (
	"github.com/pingcap/errors"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/tiflash"
)

// Ops knows how to operate TiDB with binlog on k8s
type Ops struct {
	cli     client.Client
	tiflash *tiflash.Ops
	*tidb.TidbOps
}

// New creates ops
func New(cli client.Client, tidbClient *tidb.TidbOps) *Ops {
	return &Ops{cli: cli, tiflash: tiflash.New(cli, tidbClient), TidbOps: tidbClient}
}

// Apply TiFlash abtest cluster
func (t *Ops) Apply(tc *Recommendation) error {
	var g errgroup.Group
	g.Go(func() error {
		return t.tiflash.Apply(tc.Cluster1)
	})
	g.Go(func() error {
		return t.ApplyTiDBCluster(tc.Cluster2)
	})

	return g.Wait()
}

// Delete TiFlash abtest cluster
func (t *Ops) Delete(tc *Recommendation) error {
	var g errgroup.Group
	g.Go(func() error {
		return t.tiflash.Delete(tc.Cluster1)
	})

	g.Go(func() error {
		return t.TidbOps.Delete(tc.Cluster2)
	})
	return g.Wait()
}

// GetNodes get all nodes in this cluster
func (t *Ops) GetNodes(tc *Recommendation) ([]clusterTypes.Node, error) {
	var nodes []clusterTypes.Node

	cluster1Nodes, err := t.tiflash.GetNodes(tc.Cluster1)
	if err != nil {
		return nodes, err
	}
	cluster2Nodes, err := t.TidbOps.GetNodes(tc.Cluster2)
	if err != nil {
		return nodes, err
	}

	return append(cluster1Nodes, cluster2Nodes...), nil
}

// GetClientNodes get all client nodes
func (t *Ops) GetClientNodes(tc *Recommendation) ([]clusterTypes.ClientNode, error) {
	var clientNodes []clusterTypes.ClientNode

	cluster1ClientNodes, err := t.tiflash.GetClientNodes(tc.Cluster1)
	if err != nil {
		return clientNodes, err
	}
	cluster2ClientNodes, err := t.TidbOps.GetClientNodes(tc.Cluster2)
	if err != nil {
		return clientNodes, err
	}

	clientNodes = append(cluster1ClientNodes, cluster2ClientNodes...)

	if len(clientNodes) != 2 {
		return clientNodes, errors.New("clientNodes count not 2")
	}

	return clientNodes, nil
}
