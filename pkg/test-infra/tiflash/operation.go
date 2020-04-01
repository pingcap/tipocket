// Copyright 2020 PingCAP, Inc.
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

package tiflash

import (
	"context"
	"time"

	"github.com/ngaut/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tipocket/pkg/test-infra/tests"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
)

// Ops knows how to operate TiDBConfig and TiFlash on k8s
type Ops struct {
	cli  client.Client
	tf   *tiFlash
	ns   string
	name string
}

// New creates Ops
func New(namespace, name string) *Ops {
	return &Ops{cli: tests.TestClient.Cli, ns: namespace, name: name, tf: newTiFlash(namespace, name)}
}

// Namespace ...
func (o *Ops) Namespace() string {
	return o.ns
}

// Apply TiFlash cluster
func (o *Ops) Apply() error {
	if err := o.applyObject(o.tf.ConfigMap); err != nil {
		return err
	}
	if err := o.applyObject(o.tf.Service); err != nil {
		return err
	}
	if err := o.applyObject(o.tf.StatefulSet); err != nil {
		return err
	}

	if err := o.waitTiFlashReady(o.tf.StatefulSet, 5*time.Minute); err != nil {
		return err
	}

	return nil
}

// Delete TiFlash cluster
func (o *Ops) Delete() error {
	if err := o.cli.Delete(context.TODO(), o.tf.Service); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if err := o.cli.Delete(context.TODO(), o.tf.ConfigMap); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if err := o.cli.Delete(context.TODO(), o.tf.StatefulSet); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (o *Ops) waitTiFlashReady(st *appsv1.StatefulSet, timeout time.Duration) error {
	local := st.DeepCopy()
	log.Infof("Waiting up to %v for StatefulSet %s to have all replicas ready",
		timeout, st.Name)
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		key, err := client.ObjectKeyFromObject(local)
		if err != nil {
			return false, err
		}

		if err = o.cli.Get(context.TODO(), key, local); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			log.Errorf("error getting TiFlash statefulset: %v", err)
			return false, err
		}

		if local.Status.ReadyReplicas != *local.Spec.Replicas {
			log.Infof("TiFlash %s do not have enough ready replicas, ready: %d, desired: %d",
				local.Name, local.Status.ReadyReplicas, *local.Spec.Replicas)
			return false, nil
		}
		log.Infof("All %d replicas of TiFlash %s are ready.", local.Status.ReadyReplicas, local.Name)
		return true, nil
	})
}

func (o *Ops) applyObject(object runtime.Object) error {
	if err := o.cli.Create(context.TODO(), object); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

// GetClientNodes returns the client nodes
func (o *Ops) GetClientNodes() ([]clusterTypes.ClientNode, error) {
	return nil, nil
}

// GetNodes returns all nodes
func (o *Ops) GetNodes() ([]clusterTypes.Node, error) {
	pods := &corev1.PodList{}
	if err := o.cli.List(context.TODO(), pods, &client.ListOptions{Namespace: o.ns},
		client.MatchingLabels{"app.kubernetes.io/instance": o.name + "-tiflash"}); err != nil {
		return []clusterTypes.Node{}, err
	}

	nodes := make([]clusterTypes.Node, 0, len(pods.Items))
	for _, pod := range pods.Items {
		nodes = append(nodes, clusterTypes.Node{
			Namespace: pod.ObjectMeta.Namespace,
			PodName:   pod.ObjectMeta.Name,
			IP:        pod.Status.PodIP,
			Component: clusterTypes.TiFlash,
			Port:      util.FindPort(pod.ObjectMeta.Name, pod.Spec.Containers[0].Ports),
		})
	}

	return nodes, nil
}
