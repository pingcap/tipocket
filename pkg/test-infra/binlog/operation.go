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

package binlog

import (
	"context"
	"fmt"
	"time"

	"github.com/pingcap/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
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

// Apply binlog cluster
func (t *Ops) Apply(tc *ClusterRecommendation) error {
	if err := t.ApplyTiDBCluster(tc.Upstream); err != nil {
		return err
	}

	if err := t.ApplyTiDBCluster(tc.Downstream); err != nil {
		return err
	}

	if err := t.ApplyDrainer(tc.Drainer); err != nil {
		return err
	}
	return nil
}

// Delete binlog cluster
func (t *Ops) Delete(tc *ClusterRecommendation) error {
	return nil
}

// ApplyTiDBCluster applies a cluster
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

// ApplyDrainer applies drainer
func (t *Ops) ApplyDrainer(drainer *Drainer) error {
	// apply configmap
	if err := t.ApplyObject(drainer.ConfigMap); err != nil {
		return err
	}
	// apply service
	if err := t.ApplyObject(drainer.Service); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	// apply statefulset
	if err := t.ApplyObject(drainer.StatefulSet); err != nil {
		return err
	}
	return nil
}

// ApplyObject applies object
func (t *Ops) ApplyObject(object runtime.Object) error {
	err := t.cli.Create(context.TODO(), object)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (t *Ops) GetDrainerNode(d *Drainer) (clusterTypes.Node, error) {
	pod := &corev1.Pod{}
	err := t.cli.Get(context.Background(), client.ObjectKey{
		Namespace: d.StatefulSet.ObjectMeta.Namespace,
		Name:      fmt.Sprintf("%s-0", d.StatefulSet.ObjectMeta.Name),
	}, pod)

	if err != nil {
		return clusterTypes.Node{}, err
	}

	return clusterTypes.Node{
		Namespace: pod.ObjectMeta.Namespace,
		PodName:   pod.ObjectMeta.Name,
		IP:        pod.Status.PodIP,
		Component: clusterTypes.Drainer,
		Port:      util.FindPort(pod.ObjectMeta.Name, pod.Spec.Containers[0].Ports),
	}, nil
}

func (t *Ops) GetNodes(tc *ClusterRecommendation) ([]clusterTypes.Node, error) {
	var nodes []clusterTypes.Node

	upstreamNodes, err := t.tidbClient.GetNodes(tc.Upstream)
	if err != nil {
		return nodes, err
	}
	downstreamNodes, err := t.tidbClient.GetNodes(tc.Downstream)
	if err != nil {
		return nodes, err
	}
	drainerNode, err := t.GetDrainerNode(tc.Drainer)
	if err != nil {
		return nodes, err
	}

	return append(append(upstreamNodes, downstreamNodes...), drainerNode), nil
}

func (t *Ops) GetClientNodes(tc *ClusterRecommendation) ([]clusterTypes.ClientNode, error) {
	var clientNodes []clusterTypes.ClientNode
	upstreamClientNodes, err := t.tidbClient.GetClientNodes(tc.Upstream)
	if err != nil {
		return clientNodes, err
	}
	downstreamClientNodes, err := t.tidbClient.GetClientNodes(tc.Downstream)
	if err != nil {
		return clientNodes, err
	}
	clientNodes = append(upstreamClientNodes, downstreamClientNodes...)

	if len(clientNodes) != 2 {
		return clientNodes, errors.New("clientNodes count not 2")
	}

	return clientNodes, nil
}
