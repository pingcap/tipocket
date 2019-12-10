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

package tidb

import (
	"context"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tipocket/pkg/fixture"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TidbOps knows how to operate TiDB on k8s
type TidbOps struct {
	cli client.Client
}

func New(cli client.Client) *TidbOps {
	return &TidbOps{cli}
}

func (t *TidbOps) ApplyTiDBCluster(tc *v1alpha1.TidbCluster) error {
	desired := tc.DeepCopy()
	if tc.Spec.Version == "" {
		tc.Spec.Version = fixture.E2eContext.TiDBVersion
	}
	_, err := controllerruntime.CreateOrUpdate(context.TODO(), t.cli, tc, func() error {
		tc.Spec = desired.Spec
		tc.Annotations = desired.Annotations
		tc.Labels = desired.Labels
		return nil
	})
	return err
}

func (t *TidbOps) WaitTiDBClusterReady(tc *v1alpha1.TidbCluster, timeout time.Duration) error {
	local := tc.DeepCopy()
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		key, err := client.ObjectKeyFromObject(local)
		if err != nil {
			return false, err
		}
		err = t.cli.Get(context.TODO(), key, local)
		if errors.IsNotFound(err) {
			return false, err
		}
		if err != nil {
			klog.Warningf("error getting tidbcluster: %v", err)
			return false, nil
		}
		pdReady, pdDesired := local.Status.PD.StatefulSet.ReadyReplicas, local.Spec.PD.Replicas
		if pdReady < pdDesired {
			klog.V(4).Info("PD do not have enough ready replicas, ready: %d, desired: %d", pdReady, pdDesired)
			return false, nil
		}
		tikvReady, tikvDesired := local.Status.TiKV.StatefulSet.ReadyReplicas, local.Spec.TiKV.Replicas
		if tikvReady < tikvDesired {
			klog.V(4).Info("TiKV do not have enough ready replicas, ready: %d, desired: %d", tikvReady, tikvDesired)
			return false, nil
		}
		tidbReady, tidbDesired := local.Status.TiDB.StatefulSet.ReadyReplicas, local.Spec.TiDB.Replicas
		if tidbReady < tidbDesired {
			klog.V(4).Info("TiDB do not have enough ready replicas, ready: %d, desired: %d", tikvReady, tikvDesired)
			return false, nil
		}
		return true, nil
	})
}

func (t *TidbOps) DeleteTiDBCluster(tc *v1alpha1.TidbCluster) error {
	return t.cli.Delete(context.TODO(), tc)
}

func (t *TidbOps) DeployDrainer(source *v1alpha1.TidbCluster, target string) (*appsv1.StatefulSet, error) {
	// TODO: implement
	return nil, nil
}

func (t *TidbOps) DeleteDrainer(drainer appsv1.StatefulSet) error {
	// TODO: implement
	return nil
}
