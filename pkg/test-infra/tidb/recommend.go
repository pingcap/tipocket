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

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type Recommendation struct {
	TidbCluster *v1alpha1.TidbCluster
	TidbMonitor *v1alpha1.TidbMonitor
	*corev1.Service
	NS   string
	Name string
}

func (t *Recommendation) Make() *v1alpha1.TidbCluster {
	return t.TidbCluster
}

func (t *Recommendation) EnablePump(replicas int32) *Recommendation {
	if t.TidbCluster.Spec.Pump == nil {
		t.TidbCluster.Spec.Pump = &v1alpha1.PumpSpec{
			Replicas:             replicas,
			BaseImage:            "pingcap/tidb-binlog",
			ResourceRequirements: fixture.Medium,
		}
	}
	return t
}

func (t *Recommendation) PDReplicas(replicas int32) *Recommendation {
	t.TidbCluster.Spec.PD.Replicas = replicas
	return t
}

func (t *Recommendation) TiKVReplicas(replicas int32) *Recommendation {
	t.TidbCluster.Spec.TiKV.Replicas = replicas
	return t
}

func (t *Recommendation) TiDBReplicas(replicas int32) *Recommendation {
	t.TidbCluster.Spec.TiDB.Replicas = replicas
	return t
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

// RecommendedTiDBCluster does a recommendation, tidb-operator do not have same defaults yet
func RecommendedTiDBCluster(ns, name, version string) *Recommendation {
	enablePVReclaim, exposeStatus := true, true

	return &Recommendation{
		NS:   ns,
		Name: name,
		TidbCluster: &v1alpha1.TidbCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels: map[string]string{
					"app":      "tipocket-tidbcluster",
					"instance": name,
				},
			},
			Spec: v1alpha1.TidbClusterSpec{
				PVReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
				EnablePVReclaim: &enablePVReclaim,
				ImagePullPolicy: corev1.PullAlways,
				PD: v1alpha1.PDSpec{
					Replicas:             3,
					ResourceRequirements: fixture.WithStorage(fixture.Medium, "10Gi"),
					StorageClassName:     &fixture.Context.LocalVolumeStorageClass,
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: buildImage("pd", version),
					},
				},
				TiKV: v1alpha1.TiKVSpec{
					Replicas:             int32(fixture.Context.TiKVReplicas),
					ResourceRequirements: fixture.WithStorage(fixture.Large, "200Gi"),
					StorageClassName:     &fixture.Context.LocalVolumeStorageClass,
					// disable auto fail over
					MaxFailoverCount: pointer.Int32Ptr(int32(0)),
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: buildImage("tikv", version),
					},
				},
				TiDB: v1alpha1.TiDBSpec{
					Replicas:             2,
					ResourceRequirements: fixture.Large,
					Service: &v1alpha1.TiDBServiceSpec{
						ServiceSpec: v1alpha1.ServiceSpec{
							Type: corev1.ServiceTypeNodePort,
						},
						ExposeStatus: &exposeStatus,
					},
					// disable auto fail over
					MaxFailoverCount: pointer.Int32Ptr(int32(0)),
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: buildImage("tidb", version),
					},
				},
			},
		},
		TidbMonitor: &v1alpha1.TidbMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels: map[string]string{
					"app":      "tipocket-tidbmonitor",
					"instance": name,
				},
			},
			Spec: v1alpha1.TidbMonitorSpec{
				Clusters: []v1alpha1.TidbClusterRef{
					{
						Namespace: ns,
						Name:      name,
					},
				},
				Persistent: false,
				Prometheus: v1alpha1.PrometheusSpec{
					MonitorContainer: v1alpha1.MonitorContainer{
						BaseImage: "prom/prometheus",
						Version:   "v2.11.1",
					},
					LogLevel: "info",
				},
				Grafana: &v1alpha1.GrafanaSpec{
					Service: v1alpha1.ServiceSpec{
						Type: corev1.ServiceType(fixture.Context.TiDBMonitorSvcType),
					},
					MonitorContainer: v1alpha1.MonitorContainer{
						BaseImage: "grafana/grafana",
						Version:   "6.0.1",
					},
				},
				Initializer: v1alpha1.InitializerSpec{
					MonitorContainer: v1alpha1.MonitorContainer{
						BaseImage: "pingcap/tidb-monitor-initializer",
						Version:   "v3.0.5",
					},
				},
				Reloader: v1alpha1.ReloaderSpec{
					MonitorContainer: v1alpha1.MonitorContainer{
						BaseImage: "pingcap/tidb-monitor-reloader",
						Version:   "v1.0.1",
					},
				},
				ImagePullPolicy: corev1.PullIfNotPresent,
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
