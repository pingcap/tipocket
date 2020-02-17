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
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	if err := t.ApplyDrainer(tc.Drainer); err != nil {
		return err
	}

	if err := t.ApplyTiDBCluster(tc.Downstream); err != nil {
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
	if err := t.tidbClient.WaitTiDBClusterReady(tc.TidbCluster, 10*time.Minute); err != nil {
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

	time.Sleep(10 * time.Second)

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
