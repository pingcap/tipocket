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
	"fmt"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"
	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ops knows how to operate TiDB TiFlash on k8s
type Ops struct {
	cli client.Client
	*tidb.TidbOps
}

// New creates TiFlash ops
func New(cli client.Client, tidbClient *tidb.TidbOps) *Ops {
	return &Ops{cli, tidbClient}
}

// Apply TiFlash cluster
func (o *Ops) Apply(tc *Recommendation) error {
	// config file initialization
	if fixture.Context.TiKVConfigFile == "" {
		fixture.Context.TiKVConfigFile = "/config/tiflash/tikv.toml"
	}
	if fixture.Context.PDConfigFile == "" {
		fixture.Context.PDConfigFile = "/config/tiflash/pd.toml"
	}

	if err := o.ApplyTiDBCluster(tc.TiDBCluster); err != nil {
		return err
	}

	if err := o.ApplyTiFlash(tc.TiFlash); err != nil {
		return err
	}

	return nil
}

// Delete TiFlash cluster
func (o *Ops) Delete(tc *Recommendation) error {
	if err := o.TidbOps.Delete(tc.TiDBCluster); err != nil {
		return err
	}

	if err := o.DeleteTiFlash(tc.TiFlash); err != nil {
		return err
	}

	return nil
}

func (o *Ops) ApplyTiFlash(t *TiFlash) error {
	if err := o.applyObject(t.ConfigMap); err != nil {
		return err
	}
	if err := o.applyObject(t.Service); err != nil {
		return err
	}
	if err := o.applyObject(t.StatefulSet); err != nil {
		return err
	}

	if err := o.waitTiFlashReady(t.StatefulSet, 5*time.Minute); err != nil {
		return err
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

// Delete TiFlash component
func (o *Ops) DeleteTiFlash(t *TiFlash) error {
	if err := o.cli.Delete(context.TODO(), t.Service); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if err := o.cli.Delete(context.TODO(), t.ConfigMap); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if err := o.cli.Delete(context.TODO(), t.StatefulSet); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// GetClientNodes returns the client nodes
func (o *Ops) GetClientNodes(tc *Recommendation) ([]clusterTypes.ClientNode, error) {
	clientNodes, err := o.TidbOps.GetClientNodes(tc.TiDBCluster)
	if err != nil {
		return clientNodes, err
	}

	if len(clientNodes) != 1 {
		return clientNodes, errors.New("clientNodes count not 1")
	}

	return clientNodes, nil
}

// GetNodes returns all nodes
func (o *Ops) GetNodes(tc *Recommendation) ([]clusterTypes.Node, error) {
	var nodes []clusterTypes.Node

	tidbClusterNodes, err := o.TidbOps.GetNodes(tc.TiDBCluster)
	if err != nil {
		return nodes, err
	}
	tiFlashNode, err := o.GetTiFlashNode(tc.TiFlash)
	if err != nil {
		return nodes, err
	}

	return append(tidbClusterNodes, tiFlashNode), nil
}

// GetTiFlashNode returns the nodes of TiFlash
func (o *Ops) GetTiFlashNode(TiFlash *TiFlash) (clusterTypes.Node, error) {
	pod := &corev1.Pod{}
	objKey := client.ObjectKey{
		Namespace: TiFlash.StatefulSet.ObjectMeta.Namespace,
		Name:      fmt.Sprintf("%s-0", TiFlash.StatefulSet.ObjectMeta.Name),
	}
	if err := o.cli.Get(context.Background(), objKey, pod); err != nil {
		return clusterTypes.Node{}, err
	}

	return clusterTypes.Node{
		Namespace: pod.ObjectMeta.Namespace,
		PodName:   pod.ObjectMeta.Name,
		IP:        pod.Status.PodIP,
		Component: clusterTypes.TiFlash,
		Port:      util.FindPort(pod.ObjectMeta.Name, pod.Spec.Containers[0].Ports),
	}, nil
}
