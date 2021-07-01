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

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
)

// Ops knows how to operate binlog
type Ops struct {
	cli     client.Client
	drainer *Drainer
	ns      string
}

// New creates binlog ops
func New(ns, name string) *Ops {
	return &Ops{cli: tests.TestClient.Cli, ns: ns, drainer: newDrainer(ns, name)}
}

// Apply binlog cluster
func (t *Ops) Apply() error {
	// apply configmap
	if err := util.ApplyObject(t.cli, t.drainer.ConfigMap); err != nil {
		return err
	}
	// apply service
	if err := util.ApplyObject(t.cli, t.drainer.Service); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)

	// apply statefulset
	if err := util.ApplyObject(t.cli, t.drainer.StatefulSet); err != nil {
		return err
	}
	return nil
}

// Delete binlog cluster
func (t *Ops) Delete() error {
	return nil
}

// GetNodes returns all nodes(eg. pods on k8s)
func (t *Ops) GetNodes() ([]cluster.Node, error) {
	pod := &corev1.Pod{}
	err := t.cli.Get(context.Background(), client.ObjectKey{
		Namespace: t.drainer.StatefulSet.ObjectMeta.Namespace,
		Name:      fmt.Sprintf("%s-0", t.drainer.StatefulSet.ObjectMeta.Name),
	}, pod)

	if err != nil {
		return []cluster.Node{}, err
	}
	// because the drainer is managed by statefulset with a headless service,
	// we use the fqdn as the node ip
	fqdn, err := util.GetFQDNFromStsPod(pod)
	if err != nil {
		return nil, err
	}
	return []cluster.Node{{
		Namespace: pod.ObjectMeta.Namespace,
		PodName:   pod.ObjectMeta.Name,
		IP:        fqdn,
		Component: cluster.Drainer,
		Port:      util.FindPort(pod.ObjectMeta.Name, string(cluster.Drainer), pod.Spec.Containers),
	}}, nil
}

// GetClientNodes returns client nodes
func (t *Ops) GetClientNodes() ([]cluster.ClientNode, error) {
	return nil, nil
}
