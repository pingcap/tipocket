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

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Drainer components
type Drainer struct {
	*corev1.ConfigMap
	*corev1.Service
	*appsv1.StatefulSet
}

// ClusterRecommendation defines binlog cluster
type ClusterRecommendation struct {
	*Drainer
	Upstream   *tidb.TiDBClusterRecommendation
	Downstream *tidb.TiDBClusterRecommendation
	NS         string
	Name       string
}

// RecommendedBinlogCluster create cluster with binlog
func RecommendedBinlogCluster(ns, name string) *ClusterRecommendation {
	var (
		enableBinlog             = true
		upstream                 = tidb.RecommendedTiDBCluster(ns, fmt.Sprintf("%s-upstream", name))
		downstream               = tidb.RecommendedTiDBCluster(ns, fmt.Sprintf("%s-downstream", name))
		drainerName              = fmt.Sprintf("%s-drainer", name)
		drainerReplicas    int32 = 1
		drainerServiceName       = drainerName
		drainerConfigModel       = DrainerConfigModel{
			PDAddress:    fmt.Sprintf("%s-upstream-pd", name),
			DownStreamDB: fmt.Sprintf("%s-downstream-tidb", name),
		}
		drainerCommandModel = DrainerCommandModel{
			Component: drainerName,
		}
		configmapMountMode int32 = 420
	)

	if fixture.Context.BinlogConfig.EnableRelayLog {
		drainerConfigModel.RelayPath = "/data/relay"
	}

	drainerConfigmap, _ := RenderDrainerConfig(&drainerConfigModel)
	drainerCommand, _ := RenderDrainerCommand(&drainerCommandModel)

	upstream.TidbCluster.Spec.TiDB.BinlogEnabled = &enableBinlog
	upstream.TidbCluster.Spec.Pump = &v1alpha1.PumpSpec{
		Replicas:             3,
		ResourceRequirements: fixture.WithStorage(fixture.Small, "10Gi"),
		StorageClassName:     &fixture.Context.LocalVolumeStorageClass,
		ComponentSpec: v1alpha1.ComponentSpec{
			Image: buildBinlogImage("tidb-binlog"),
		},
		GenericConfig: config.GenericConfig{
			Config: map[string]interface{}{},
		},
	}

	return &ClusterRecommendation{
		NS:         ns,
		Name:       name,
		Upstream:   upstream,
		Downstream: downstream,
		Drainer: &Drainer{
			&corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: ns,
					Name:      drainerName,
				},
				Data: map[string]string{
					"drainer-config": drainerConfigmap,
				},
			},
			&corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s-drainer", name),
					Namespace: ns,
				},
				Spec: corev1.ServiceSpec{
					Type: "ClusterIP",
					Ports: []corev1.ServicePort{
						{
							Name: "drainer",
							Port: 8249,
						},
					},
					ClusterIP: "",
					Selector: map[string]string{
						"app.kubernetes.io/name":      "tidb-cluster",
						"app.kubernetes.io/component": "drainer",
						"app.kubernetes.io/instance":  drainerName,
					},
				},
			},
			&appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      drainerName,
					Namespace: ns,
					Labels: map[string]string{
						"app":      "tipocket-tidbcluster",
						"instance": "name",
					},
				},
				Spec: appsv1.StatefulSetSpec{
					ServiceName: drainerServiceName,
					Replicas:    &drainerReplicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component": "drainer",
							"app.kubernetes.io/instance":  drainerName,
							"app.kubernetes.io/name":      "tidb-cluster",
						},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/component": "drainer",
								"app.kubernetes.io/instance":  drainerName,
								"app.kubernetes.io/name":      "tidb-cluster",
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:            "drainer",
									Image:           buildBinlogImage("tidb-binlog"),
									ImagePullPolicy: "IfNotPresent",
									Command: []string{
										"/bin/sh",
										"-c",
										drainerCommand,
									},
									Ports: []corev1.ContainerPort{
										{
											Name:          "drainer",
											ContainerPort: 8249,
										},
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "data",
											MountPath: "/data",
										},
										{
											Name:      "config",
											MountPath: "/etc/drainer",
										},
									},
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "config",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: drainerName,
											},
											DefaultMode: &configmapMountMode,
											Items: []corev1.KeyToPath{
												{
													Key:  "drainer-config",
													Path: "drainer.toml",
												},
											},
										},
									},
								},
							},
						},
					},
					VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name: "data",
							},
							Spec: corev1.PersistentVolumeClaimSpec{
								AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
								StorageClassName: &fixture.Context.LocalVolumeStorageClass,
								Resources:        fixture.WithStorage(fixture.Medium, "10Gi"),
							},
						},
					},
				},
			},
		},
	}
}

func buildBinlogImage(name string) string {
	var (
		b       strings.Builder
		version = fixture.Context.ImageVersion
	)

	if fixture.Context.BinlogConfig.BinlogVersion != "" {
		version = fixture.Context.BinlogConfig.BinlogVersion
	}
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
