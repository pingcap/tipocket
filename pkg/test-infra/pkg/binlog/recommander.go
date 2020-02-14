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
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util/config"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"
)

// ClusterRecommendation defines binlog cluster
type ClusterRecommendation struct {
	Upstream   *tidb.TiDBClusterRecommendation
	Downstream *tidb.TiDBClusterRecommendation
}

// RecommendedBinlogCluster create cluster with binlog
func RecommendedBinlogCluster(ns, name string) *ClusterRecommendation {
	var (
		upstream   = tidb.RecommendedTiDBCluster(ns, fmt.Sprintf("%s-upstream", name))
		downstream = tidb.RecommendedTiDBCluster(ns, fmt.Sprintf("%s-downstream", name))
	)
	// upstream.TidbCluster.Spec.TiDB.BinlogEnabled = true
	upstream.TidbCluster.Spec.Pump = &v1alpha1.PumpSpec{
		Replicas:             3,
		ResourceRequirements: fixture.WithStorage(fixture.Small, "10Gi"),
		StorageClassName:     &fixture.E2eContext.LocalVolumeStorageClass,
		ComponentSpec: v1alpha1.ComponentSpec{
			Image: buildImage("tidb-binlog"),
		},
		GenericConfig: config.GenericConfig{
			Config: map[string]interface{}{},
		},
	}

	return &ClusterRecommendation{
		Upstream:   upstream,
		Downstream: downstream,
	}
}

func buildImage(name string) string {
	var b strings.Builder
	if fixture.E2eContext.HubAddress != "" {
		fmt.Fprintf(&b, "%s/", fixture.E2eContext.HubAddress)
	}
	b.WriteString(fixture.E2eContext.DockerRepository)
	b.WriteString("/")
	b.WriteString(name)
	b.WriteString(":")
	b.WriteString(fixture.E2eContext.ImageVersion)
	return b.String()
}
