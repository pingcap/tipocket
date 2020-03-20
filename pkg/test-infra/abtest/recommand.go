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

package abtest

import (
	"fmt"
	"strings"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
)

// Recommendation defines 2 clusters for abtest
type Recommendation struct {
	Cluster1 *tidb.TiDBClusterRecommendation
	Cluster2 *tidb.TiDBClusterRecommendation
	NS       string
	Name     string
}

// RecommendedCluster gives recommand cluster
func RecommendedCluster(ns, name string) *Recommendation {
	r := Recommendation{
		Cluster1: tidb.RecommendedTiDBCluster(ns, fmt.Sprintf("%s-a", name)),
		Cluster2: tidb.RecommendedTiDBCluster(ns, fmt.Sprintf("%s-b", name)),
		NS:       ns,
		Name:     name,
	}

	overwriteClusterVeresion(fixture.Context.ABTestConfig.Cluster1Version, r.Cluster1)
	overwriteClusterVeresion(fixture.Context.ABTestConfig.Cluster2Version, r.Cluster2)

	return &r
}

func overwriteClusterVeresion(version string, tc *tidb.TiDBClusterRecommendation) {
	if version == "" {
		return
	}

	tc.TidbCluster.Spec.Version = version
	tc.TidbCluster.Spec.TiDB.ComponentSpec.Version = &version
	tc.TidbCluster.Spec.TiDB.ComponentSpec.Image = buildImage("tidb", version)
	tc.TidbCluster.Spec.TiKV.ComponentSpec.Version = &version
	tc.TidbCluster.Spec.TiKV.ComponentSpec.Image = buildImage("tikv", version)
	tc.TidbCluster.Spec.PD.ComponentSpec.Version = &version
	tc.TidbCluster.Spec.PD.ComponentSpec.Image = buildImage("pd", version)
}

func buildImage(name, version string) string {
	var b strings.Builder
	if fixture.Context.HubAddress != "" {
		fmt.Fprintf(&b, "%s/", fixture.Context.HubAddress)
	}
	b.WriteString(fixture.Context.DockerRepository)
	b.WriteString("/")
	b.WriteString(name)
	b.WriteString(":")
	b.WriteString(version)
	return b.String()
}
