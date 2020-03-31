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

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
)

// Ops knows how to operate TiDB with binlog on k8s
type Ops struct {
	cli     client.Client
	drainer *Drainer
}

// New creates binlog ops
func New(cli client.Client, tidbClient *tidb.Ops) *Ops {
	return &Ops{cli, tidbClient}
}

// Apply binlog cluster
<<<<<<< HEAD
func (o *Ops) Apply(tc *Recommendation) error {
	if err := o.TidbOps.ApplyTiDBCluster(tc.Upstream); err != nil {
		return err
	}

	if err := o.TidbOps.ApplyTiDBCluster(tc.Downstream); err != nil {
		return err
	}

	if err := o.ApplyDrainer(tc.Drainer); err != nil {
		return err
	}
	return nil
}

// Delete binlog cluster
func (o *Ops) Delete(tc *Recommendation) error {
	if err := o.TidbOps.Delete(tc.Upstream); err != nil {
		return err
	}
	// TODO: delete drainer
	if err := o.TidbOps.Delete(tc.Downstream); err != nil {
		return err
	}
	return nil
}

// ApplyDrainer applies drainer
func (o *Ops) ApplyDrainer(drainer *Drainer) error {
	// apply configmap
	if err := o.ApplyObject(drainer.ConfigMap); err != nil {
		return err
	}
	// apply service
	if err := o.ApplyObject(drainer.Service); err != nil {
=======
func (t *Ops) Apply() error {
	// apply configmap
	if err := util.ApplyObject(t.cli, t.drainer.ConfigMap); err != nil {
		return err
	}
	// apply service
	if err := util.ApplyObject(t.cli, t.drainer.Service); err != nil {
>>>>>>> 307de7e... refactore recommendation to an interface
		return err
	}

	time.Sleep(5 * time.Second)

	// apply statefulset
<<<<<<< HEAD
	if err := o.ApplyObject(drainer.StatefulSet); err != nil {
=======
	if err := util.ApplyObject(t.cli, t.drainer.StatefulSet); err != nil {
>>>>>>> 307de7e... refactore recommendation to an interface
		return err
	}
	return nil
}

<<<<<<< HEAD
// ApplyObject applies object
func (o *Ops) ApplyObject(object runtime.Object) error {
	err := o.cli.Create(context.TODO(), object)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// GetDrainerNode ...
func (o *Ops) GetDrainerNode(d *Drainer) (clusterTypes.Node, error) {
	pod := &corev1.Pod{}
	err := o.cli.Get(context.Background(), client.ObjectKey{
		Namespace: d.StatefulSet.ObjectMeta.Namespace,
		Name:      fmt.Sprintf("%s-0", d.StatefulSet.ObjectMeta.Name),
=======
// Delete binlog cluster
func (t *Ops) Delete() error {
	return nil
}

func (t *Ops) GetNodes() ([]clusterTypes.Node, error) {
	pod := &corev1.Pod{}
	err := t.cli.Get(context.Background(), client.ObjectKey{
		Namespace: t.drainer.StatefulSet.ObjectMeta.Namespace,
		Name:      fmt.Sprintf("%s-0", t.drainer.StatefulSet.ObjectMeta.Name),
>>>>>>> 307de7e... refactore recommendation to an interface
	}, pod)

	if err != nil {
		return []clusterTypes.Node{}, err
	}

	return []clusterTypes.Node{clusterTypes.Node{
		Namespace: pod.ObjectMeta.Namespace,
		PodName:   pod.ObjectMeta.Name,
		IP:        pod.Status.PodIP,
		Component: clusterTypes.Drainer,
		Port:      util.FindPort(pod.ObjectMeta.Name, pod.Spec.Containers[0].Ports),
	}}, nil
}

<<<<<<< HEAD
// GetNodes ...
func (o *Ops) GetNodes(tc *Recommendation) ([]clusterTypes.Node, error) {
	var nodes []clusterTypes.Node

	upstreamNodes, err := o.TidbOps.GetNodes(tc.Upstream)
	if err != nil {
		return nodes, err
	}
	downstreamNodes, err := o.TidbOps.GetNodes(tc.Downstream)
	if err != nil {
		return nodes, err
	}
	drainerNode, err := o.GetDrainerNode(tc.Drainer)
	if err != nil {
		return nodes, err
	}

	return append(append(upstreamNodes, downstreamNodes...), drainerNode), nil
}

// GetClientNodes ...
func (o *Ops) GetClientNodes(tc *Recommendation) ([]clusterTypes.ClientNode, error) {
	var clientNodes []clusterTypes.ClientNode
	upstreamClientNodes, err := o.TidbOps.GetClientNodes(tc.Upstream)
	if err != nil {
		return clientNodes, err
	}
	downstreamClientNodes, err := o.TidbOps.GetClientNodes(tc.Downstream)
	if err != nil {
		return clientNodes, err
	}
	clientNodes = append(upstreamClientNodes, downstreamClientNodes...)

	if len(clientNodes) != 2 {
		return clientNodes, errors.New("clientNodes count not 2")
	}

	return clientNodes, nil
=======
func (t *Ops) GetClientNodes() ([]clusterTypes.ClientNode, error) {
	return nil, nil
>>>>>>> 307de7e... refactore recommendation to an interface
}
