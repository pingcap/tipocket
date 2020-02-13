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
	"fmt"
	"strings"

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
)

type TiDBClusterRecommendation struct {
	*v1alpha1.TidbCluster
	*corev1.Service
}

func (t *TiDBClusterRecommendation) Make() *v1alpha1.TidbCluster {
	return t.TidbCluster
}

func (t *TiDBClusterRecommendation) EnablePump(replicas int32) *TiDBClusterRecommendation {
	if t.TidbCluster.Spec.Pump == nil {
		t.TidbCluster.Spec.Pump = &v1alpha1.PumpSpec{
			Replicas: replicas,
			ComponentSpec: v1alpha1.ComponentSpec{
				BaseImage: "pingcap/tidb-binlog",
			},
			Resources: fixture.Medium,
		}
	}
	return t
}

func (t *TiDBClusterRecommendation) PDReplicas(replicas int32) *TiDBClusterRecommendation {
	t.TidbCluster.Spec.PD.Replicas = replicas
	return t
}

func (t *TiDBClusterRecommendation) TiKVReplicas(replicas int32) *TiDBClusterRecommendation {
	t.TidbCluster.Spec.TiKV.Replicas = replicas
	return t
}

func (t *TiDBClusterRecommendation) TiDBReplicas(replicas int32) *TiDBClusterRecommendation {
	t.TidbCluster.Spec.TiDB.Replicas = replicas
	return t
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

// RecommendedTiDBCluster does a recommendation, tidb-operator do not have same defaults yet
func RecommendedTiDBCluster(ns, name string) *TiDBClusterRecommendation {
	return &TiDBClusterRecommendation{
		TidbCluster: &v1alpha1.TidbCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels: map[string]string{
					"app":      "e2e-tidbcluster",
					"instance": "name",
				},
			},
			Spec: v1alpha1.TidbClusterSpec{
				Version:         fixture.E2eContext.TiDBVersion,
				PVReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
				EnablePVReclaim: true,
				PD: v1alpha1.PDSpec{
					Replicas:         3,
					Resources:        fixture.WithStorage(fixture.Small, "10Gi"),
					StorageClassName: fixture.E2eContext.LocalVolumeStorageClass,
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: buildImage("pd"),
					},
				},
				TiKV: v1alpha1.TiKVSpec{
					Replicas:         3,
					Resources:        fixture.WithStorage(fixture.Medium, "10Gi"),
					StorageClassName: fixture.E2eContext.LocalVolumeStorageClass,
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: buildImage("tikv"),
					},
				},
				TiDB: v1alpha1.TiDBSpec{
					Replicas:  2,
					Resources: fixture.Medium,
					Service: &v1alpha1.TiDBServiceSpec{
						ServiceSpec: v1alpha1.ServiceSpec{
							Type: corev1.ServiceTypeNodePort,
						},
						ExposeStatus: true,
					},
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: buildImage("tidb"),
					},
				},
			},
		},
		Service: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-tidb", name),
				Namespace: ns,
			},
			Spec: corev1.ServiceSpec{
				Type: "NodePort",
				Ports: []corev1.ServicePort{
					{
						Name: PortNameMySQLClient,
						Port: 4000,
					},
					{
						Name: PortNameStatus,
						Port: 10080,
					},
				},
				Selector: map[string]string{
					"app.kubernetes.io/component": "tidb",
					"app.kubernetes.io/name":      "tidb-cluster",
				},
				ClusterIP: "",
			},
		},
	}
}
