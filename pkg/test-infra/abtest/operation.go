// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package abtest

import (
	"github.com/pingcap/errors"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
)

// Ops knows how to operate TiDB with binlog on k8s
type Ops struct {
	cli        client.Client
	tidbClient *tidb.TidbOps
}

// New creates binlog ops
func New(cli client.Client, tidbClient *tidb.TidbOps) *Ops {
	return &Ops{cli, tidbClient}
}

// Apply abtest cluster
func (t *Ops) Apply(tc *Recommendation) error {
	var g errgroup.Group

	g.Go(func() error {
		return t.ApplyTiDBCluster(tc.Cluster1)
	})
	g.Go(func() error {
		return t.ApplyTiDBCluster(tc.Cluster2)
	})

	return g.Wait()
}

// ApplyTiDBCluster apply a tidb cluster
func (t *Ops) ApplyTiDBCluster(tc *tidb.TiDBClusterRecommendation) error {
	if err := t.tidbClient.ApplyTiDBCluster(tc.TidbCluster); err != nil {
		return err
	}
	if err := t.tidbClient.WaitTiDBClusterReady(tc.TidbCluster, fixture.Context.WaitClusterReadyDuration); err != nil {
		return err
	}
	if err := t.tidbClient.ApplyTiDBService(tc.Service); err != nil {
		return err
	}
	return nil
}

// GetNodes get all nodes in this cluster
func (t *Ops) GetNodes(tc *Recommendation) ([]clusterTypes.Node, error) {
	var nodes []clusterTypes.Node

	cluster1Nodes, err := t.tidbClient.GetNodes(tc.Cluster1)
	if err != nil {
		return nodes, err
	}
	cluster2Nodes, err := t.tidbClient.GetNodes(tc.Cluster2)
	if err != nil {
		return nodes, err
	}

	return append(cluster1Nodes, cluster2Nodes...), nil
}

// GetClientNodes get all client nodes
func (t *Ops) GetClientNodes(tc *Recommendation) ([]clusterTypes.ClientNode, error) {
	var clientNodes []clusterTypes.ClientNode

	cluster1ClientNodes, err := t.tidbClient.GetClientNodes(tc.Cluster1)
	if err != nil {
		return clientNodes, err
	}
	cluster2ClientNodes, err := t.tidbClient.GetClientNodes(tc.Cluster2)
	if err != nil {
		return clientNodes, err
	}

	clientNodes = append(cluster1ClientNodes, cluster2ClientNodes...)

	if len(clientNodes) != 2 {
		return clientNodes, errors.New("clientNodes count not 2")
	}

	return clientNodes, nil
}
