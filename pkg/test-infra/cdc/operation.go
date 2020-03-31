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

package cdc

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql" // mysql driver
	"github.com/ngaut/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/util"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CDCOps knows how to operate TiDB CDC on k8s
type CDCOps struct {
	cli client.Client
	cdc *CDC
}

// New creates cdc ops
func New(ns, name string) *CDCOps {
	return &CDCOps{cdc: newCDC(ns, name)}
}

func (c *CDCOps) Namespace() string {
	return c.cdc.Namespace
}

// Apply CDC cluster
func (c *CDCOps) Apply() error {
	if err := c.applyCDC(); err != nil {
		return err
	}

	if err := c.applyJob(); err != nil {
		return err
	}
	return nil
}

// Delete CDC cluster
func (c *CDCOps) Delete() error {
	if err := c.cli.Delete(context.TODO(), c.cdc.Service); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if err := c.cli.Delete(context.TODO(), c.cdc.StatefulSet); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func (c *CDCOps) applyCDC() error {
	if err := util.ApplyObject(c.cli, c.cdc.Service); err != nil {
		return err
	}
	if err := util.ApplyObject(c.cli, c.cdc.StatefulSet); err != nil {
		return err
	}

	if err := c.waitCDCReady(c.cdc.StatefulSet, 5*time.Minute); err != nil {
		return err
	}
	return nil
}

func (o *Ops) waitCDCReady(st *appsv1.StatefulSet, timeout time.Duration) error {
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
			log.Errorf("error getting CDC statefulset: %v", err)
			return false, err
		}

		if local.Status.ReadyReplicas != *local.Spec.Replicas {
			log.Infof("CDC %s do not have enough ready replicas, ready: %d, desired: %d",
				local.Name, local.Status.ReadyReplicas, *local.Spec.Replicas)
			return false, nil
		}
		log.Infof("All %d replicas of CDC %s are ready.", local.Status.ReadyReplicas, local.Name)
		return true, nil
	})
}

func (o *Ops) waitJobCompleted(job *batchv1.Job) error {
	local := job.DeepCopy()
	log.Infof("Waiting up to %s for job completed", local.Name)

	return wait.PollImmediate(5*time.Second, 5*time.Minute, func() (bool, error) {
		key, err := client.ObjectKeyFromObject(local)
		if err != nil {
			return false, err
		}

		if err = o.cli.Get(context.TODO(), key, local); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			log.Errorf("error getting CDC job: %v", err)
			return false, err
		}

		if local.Status.Succeeded != 1 {
			log.Infof("Job %s was not completed.", local.Name)
			return false, nil
		}

		log.Infof("Job %s has been completed.", local.Name)
		return true, nil
	})
}

func (c *CDCOps) applyJob() error {
	if err := util.ApplyObject(c.cli, c.cdc.Job); err != nil {
		return err
	}
	return c.waitJobCompleted(c.cdc.Job)
}

// GetClientNodes returns the client nodes
func (c *CDCOps) GetClientNodes() ([]clusterTypes.ClientNode, error) {
	return nil, nil
}

// GetNodes returns cdc nodes
func (c *CDCOps) GetNodes() ([]clusterTypes.Node, error) {
	pod := &corev1.Pod{}
	err := c.cli.Get(context.Background(), client.ObjectKey{
		Namespace: c.cdc.StatefulSet.ObjectMeta.Namespace,
		Name:      fmt.Sprintf("%s-0", c.cdc.StatefulSet.ObjectMeta.Name),
	}, pod)

	if err != nil {
		return []clusterTypes.Node{}, err
	}

	return []clusterTypes.Node{clusterTypes.Node{
		Namespace: pod.ObjectMeta.Namespace,
		PodName:   pod.ObjectMeta.Name,
		IP:        pod.Status.PodIP,
		Component: clusterTypes.CDC,
		Port:      util.FindPort(pod.ObjectMeta.Name, pod.Spec.Containers[0].Ports),
	}}, nil
}
