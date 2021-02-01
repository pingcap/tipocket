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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
	"github.com/pingcap/tipocket/pkg/tidb-operator/apis/pingcap/v1alpha1"
)

// Recommendation ...
type Recommendation struct {
	TidbCluster *v1alpha1.TidbCluster
	TidbMonitor *v1alpha1.TidbMonitor
	*corev1.Service
}

// EnablePump ...
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

// EnableTiFlash add TiFlash spec in TiDB cluster
func (t *Recommendation) EnableTiFlash(config fixture.TiDBClusterConfig) {
	tag, fullImage, replicas := config.ImageVersion, config.TiFlashImage, config.TiFlashReplicas
	if t.TidbCluster.Spec.TiFlash == nil {
		t.TidbCluster.Spec.TiFlash = &v1alpha1.TiFlashSpec{
			Replicas:         int32(replicas),
			MaxFailoverCount: pointer.Int32Ptr(0),
			ComponentSpec: v1alpha1.ComponentSpec{
				Image: util.BuildImage("tiflash", tag, fullImage),
			},
			StorageClaims: []v1alpha1.StorageClaim{
				{
					StorageClassName: &fixture.Context.LocalVolumeStorageClass,
					Resources: fixture.WithStorage(corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("1000m"),
							fixture.Memory: resource.MustParse("2Gi"),
						},
						Limits: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("4000m"),
							fixture.Memory: resource.MustParse("16Gi"),
						},
					}, "50Gi"),
				},
			},
		}
	}
}

// PDReplicas ...
func (t *Recommendation) PDReplicas(replicas int32) *Recommendation {
	t.TidbCluster.Spec.PD.Replicas = replicas
	return t
}

// TiKVReplicas ...
func (t *Recommendation) TiKVReplicas(replicas int32) *Recommendation {
	t.TidbCluster.Spec.TiKV.Replicas = replicas
	return t
}

// TiDBReplicas ...
func (t *Recommendation) TiDBReplicas(replicas int32) *Recommendation {
	t.TidbCluster.Spec.TiDB.Replicas = replicas
	return t
}

// RecommendedTiDBCluster does a recommendation, tidb-operator do not have same defaults yet
func RecommendedTiDBCluster(ns, name string, clusterConfig fixture.TiDBClusterConfig) *Recommendation {
	enablePVReclaim, exposeStatus := true, true
	reclaimDelete := corev1.PersistentVolumeReclaimDelete
	r := &Recommendation{
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
				Timezone:        "UTC",
				PVReclaimPolicy: &reclaimDelete,
				EnablePVReclaim: &enablePVReclaim,
				ImagePullPolicy: corev1.PullAlways,
				PD: &v1alpha1.PDSpec{
					Replicas:             int32(clusterConfig.PDReplicas),
					ResourceRequirements: fixture.WithStorage(fixture.Medium, "10Gi"),
					StorageClassName:     &fixture.Context.LocalVolumeStorageClass,
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: util.BuildImage("pd", clusterConfig.ImageVersion, clusterConfig.PDImage),
					},
					StorageVolumes: []v1alpha1.StorageVolume{
						{
							Name:        "log",
							StorageSize: "200Gi",
							MountPath:   "/var/log/pdlog",
						},
					},
				},
				TiKV: &v1alpha1.TiKVSpec{
					Replicas: int32(clusterConfig.TiKVReplicas),
					ResourceRequirements: fixture.WithStorage(corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("500m"),
							fixture.Memory: resource.MustParse("4Gi"),
						},
						Limits: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("1000m"),
							fixture.Memory: resource.MustParse("16Gi"),
						},
					}, "200Gi"),
					StorageClassName: &fixture.Context.LocalVolumeStorageClass,
					// disable auto fail over
					MaxFailoverCount: pointer.Int32Ptr(int32(0)),
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: util.BuildImage("tikv", clusterConfig.ImageVersion, clusterConfig.TiKVImage),
					},
				},
				TiDB: &v1alpha1.TiDBSpec{
					Replicas: int32(clusterConfig.TiDBReplicas),
					ResourceRequirements: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("1000m"),
							fixture.Memory: resource.MustParse("1Gi"),
						},
						Limits: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("1000m"),
							fixture.Memory: resource.MustParse("16Gi"),
						},
					},
					Service: &v1alpha1.TiDBServiceSpec{
						ServiceSpec: v1alpha1.ServiceSpec{
							Type: corev1.ServiceTypeNodePort,
						},
						ExposeStatus: &exposeStatus,
					},
					StorageClassName: &fixture.Context.LocalVolumeStorageClass,
					// disable auto fail over
					MaxFailoverCount: pointer.Int32Ptr(int32(0)),
					ComponentSpec: v1alpha1.ComponentSpec{
						Image: util.BuildImage("tidb", clusterConfig.ImageVersion, clusterConfig.TiDBImage),
					},
					StorageVolumes: []v1alpha1.StorageVolume{
						{
							Name:        "log",
							StorageSize: "200Gi",
							MountPath:   "/var/log/tidblog",
						},
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
						Version:   "v4.0.9",
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
			},
		},
	}

	if clusterConfig.TiFlashReplicas > 0 {
		r.EnableTiFlash(clusterConfig)
	}
	return r
}
