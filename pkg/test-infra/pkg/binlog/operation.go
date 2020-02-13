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
	"time"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"
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

// Delete binlog cluster
func (t *Ops) Delete(tc *ClusterRecommendation) error {
	return nil
}

// Apply binlog cluster
func (t *Ops) Apply(tc *ClusterRecommendation) error {
	if err := t.tidbClient.ApplyTiDBCluster(tc.Upstream.TidbCluster); err != nil {
		return err
	}
	if err := t.tidbClient.WaitTiDBClusterReady(tc.Upstream.TidbCluster, 10*time.Minute); err != nil {
		return err
	}
	if err := t.tidbClient.ApplyTiDBService(tc.Upstream.Service); err != nil {
		return err
	}

	if err := t.tidbClient.ApplyTiDBCluster(tc.Downstream.TidbCluster); err != nil {
		return err
	}
	if err := t.tidbClient.WaitTiDBClusterReady(tc.Downstream.TidbCluster, 10*time.Minute); err != nil {
		return err
	}
	if err := t.tidbClient.ApplyTiDBService(tc.Downstream.Service); err != nil {
		return err
	}
	return nil
}
