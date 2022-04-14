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
	"github.com/pingcap/tipocket/pkg/tidb-operator/util/config"
)

// Recommendation ...
type Recommendation struct {
	TidbCluster *v1alpha1.TidbCluster
	TidbMonitor *v1alpha1.TidbMonitor
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
func (t *Recommendation) EnableTiFlash(clusterConfig fixture.TiDBClusterConfig) {
	tag, fullImage, replicas := clusterConfig.ImageVersion, clusterConfig.TiFlashImage, clusterConfig.TiFlashReplicas
	tiflashDataStorageClass := provideTiFlashStorageClassName(clusterConfig)
	if t.TidbCluster.Spec.TiFlash == nil {
		baseImage, version := util.BuildBaseImageAndVersion("tiflash", tag, fullImage)
		t.TidbCluster.Spec.TiFlash = &v1alpha1.TiFlashSpec{
			Replicas:         int32(replicas),
			MaxFailoverCount: pointer.Int32Ptr(int32(0)),
			BaseImage:        baseImage,
			ComponentSpec: v1alpha1.ComponentSpec{
				Version: &version,
			},
			StorageClaims: []v1alpha1.StorageClaim{
				{
					StorageClassName: &tiflashDataStorageClass,
					Resources: fixture.WithStorage(corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("1000m"),
							fixture.Memory: resource.MustParse("8Gi"),
						},
						Limits: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("4000m"),
							fixture.Memory: resource.MustParse("16Gi"),
						},
					}, "100Gi"),
				},
			},
		}
		tiflashCfg := v1alpha1.NewTiFlashConfig()
		tiflashCfg.Common.Set("logger.level", "debug")
		t.TidbCluster.Spec.TiFlash.Config = tiflashCfg
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

// TiDBReplicas ...
func (t *Recommendation) defineLogStorageClassName(clusterConfig fixture.TiDBClusterConfig) {
	logSC := ""
	if len(fixture.Context.LocalVolumeStorageClass) > 0 {
		logSC = fixture.Context.LocalVolumeStorageClass
	}
	if len(clusterConfig.LogStorageClassName) > 0 {
		logSC = clusterConfig.LogStorageClassName
	}
	t.TidbCluster.Spec.PD.StorageVolumes[0].StorageClassName = &logSC
	t.TidbCluster.Spec.TiDB.StorageVolumes[0].StorageClassName = &logSC
}

func providePDStorageClassName(clusterConfig fixture.TiDBClusterConfig) string {
	sc := fixture.Context.LocalVolumeStorageClass
	if len(fixture.Context.TiDBClusterConfig.PDStorageClassName) > 0 {
		sc = fixture.Context.TiDBClusterConfig.PDStorageClassName
	}
	if len(clusterConfig.PDStorageClassName) > 0 {
		sc = clusterConfig.PDStorageClassName
	}
	return sc
}

func provideTiKVStorageClassName(clusterConfig fixture.TiDBClusterConfig) string {
	sc := fixture.Context.LocalVolumeStorageClass
	if len(fixture.Context.TiDBClusterConfig.TiKVStorageClassName) > 0 {
		sc = fixture.Context.TiDBClusterConfig.TiKVStorageClassName
	}
	if len(clusterConfig.TiKVStorageClassName) > 0 {
		sc = clusterConfig.TiKVStorageClassName
	}
	return sc
}

func provideTiFlashStorageClassName(clusterConfig fixture.TiDBClusterConfig) string {
	sc := fixture.Context.LocalVolumeStorageClass
	if len(fixture.Context.TiDBClusterConfig.TiFlashStorageClassName) > 0 {
		sc = fixture.Context.TiDBClusterConfig.TiFlashStorageClassName
	}
	if len(clusterConfig.TiFlashStorageClassName) > 0 {
		sc = clusterConfig.TiFlashStorageClassName
	}
	return sc
}

// RecommendedTiDBCluster does a recommendation, tidb-operator do not have same defaults yet
func RecommendedTiDBCluster(ns, name string, clusterConfig fixture.TiDBClusterConfig) *Recommendation {
	enablePVReclaim, exposeStatus := true, true
	reclaimDelete := corev1.PersistentVolumeReclaimDelete
	pdDataStorageClass := providePDStorageClassName(clusterConfig)
	tikvDataStorageClass := provideTiKVStorageClassName(clusterConfig)

	pdBaseImage, pdVersion := util.BuildBaseImageAndVersion("pd", clusterConfig.ImageVersion, clusterConfig.PDImage)
	tikvBaseImage, tikvVersion := util.BuildBaseImageAndVersion("tikv", clusterConfig.ImageVersion, clusterConfig.TiKVImage)
	tidbBaseImage, tidbVersion := util.BuildBaseImageAndVersion("tidb", clusterConfig.ImageVersion, clusterConfig.TiDBImage)
	ticdcBaseImage, ticdcVersion := util.BuildBaseImageAndVersion("ticdc", clusterConfig.ImageVersion, clusterConfig.TiCDCImage)
	version := tidbVersion

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
				Version:         version,
				PD: &v1alpha1.PDSpec{
					Replicas:             int32(clusterConfig.PDReplicas),
					ResourceRequirements: fixture.WithStorage(fixture.Medium, "10Gi"),
					StorageClassName:     &pdDataStorageClass,
					BaseImage:            pdBaseImage,
					ComponentSpec: v1alpha1.ComponentSpec{
						Version: &pdVersion,
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
							fixture.Memory: resource.MustParse("1Gi"),
						},
						Limits: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("1000m"),
							fixture.Memory: resource.MustParse("16Gi"),
						},
					}, "200Gi"),
					StorageClassName: &tikvDataStorageClass,
					// disable auto fail over
					MaxFailoverCount: pointer.Int32Ptr(int32(0)),
					BaseImage:        tikvBaseImage,
					ComponentSpec: v1alpha1.ComponentSpec{
						Version: &tikvVersion,
					},
				},
				TiDB: &v1alpha1.TiDBSpec{
					Replicas: int32(clusterConfig.TiDBReplicas),
					ResourceRequirements: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("500m"),
							fixture.Memory: resource.MustParse("512Mi"),
						},
						Limits: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("8000m"),
							fixture.Memory: resource.MustParse("16Gi"),
						},
					},
					Service: &v1alpha1.TiDBServiceSpec{
						ServiceSpec: v1alpha1.ServiceSpec{
							Type: corev1.ServiceTypeNodePort,
						},
						ExposeStatus: &exposeStatus,
					},
					// disable auto fail over
					MaxFailoverCount: pointer.Int32Ptr(int32(0)),
					BaseImage:        tidbBaseImage,
					ComponentSpec: v1alpha1.ComponentSpec{
						Version: &tidbVersion,
					},
					StorageVolumes: []v1alpha1.StorageVolume{
						{
							Name:        "log",
							StorageSize: "200Gi",
							MountPath:   "/var/log/tidblog",
						},
					},
				},
				TiCDC: &v1alpha1.TiCDCSpec{
					Replicas:  int32(clusterConfig.TiCDCReplicas),
					BaseImage: ticdcBaseImage,
					ComponentSpec: v1alpha1.ComponentSpec{
						Version: &ticdcVersion,
					},
					ResourceRequirements: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("1000m"),
							fixture.Memory: resource.MustParse("1Gi"),
						},
						Limits: corev1.ResourceList{
							fixture.CPU:    resource.MustParse("1000m"),
							fixture.Memory: resource.MustParse("8Gi"),
						},
					},
					Config: &v1alpha1.CDCConfigWraper{
						GenericConfig: &config.GenericConfig{
							MP: map[string]interface{}{
								"timezone": fixture.Context.CDCConfig.Timezone,
								"logLevel": fixture.Context.CDCConfig.LogLevel,
								"logFile":  fixture.Context.CDCConfig.LogFile,
							},
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
						Version:   "nightly",
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
	}
	if clusterConfig.TiFlashReplicas > 0 {
		r.EnableTiFlash(clusterConfig)
	}
	r.defineLogStorageClassName(clusterConfig)
	if clusterConfig.Ref != nil {
		r.TidbCluster.Spec.PDAddresses = []string{
			fmt.Sprintf("http://%s-pd-0.%s-pd-peer.%s.svc:2379",
				clusterConfig.Ref.Name, clusterConfig.Ref.Name, clusterConfig.Ref.Namespace),
		}
		r.TidbMonitor = nil
	}
	return r
}
