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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pingcap/tipocket/pkg/test-infra/util"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

// Drainer components
type Drainer struct {
	*corev1.ConfigMap
	*corev1.Service
	*appsv1.StatefulSet
}

// RecommendedBinlogCluster creates cluster with binlog
func newDrainer(ns, name string) *Drainer {
	var (
		drainerName              = fmt.Sprintf("%s-drainer", name)
		drainerReplicas    int32 = 1
		drainerServiceName       = drainerName
		drainerConfigModel       = DrainerConfigModel{
			PDAddress:    fmt.Sprintf("%s-upstream-pd", name),
			DownStreamDB: fmt.Sprintf("%s-downstream-tidb", name),
		}
		drainerCommandModel = DrainerCommandModel{
			Component:   drainerName,
			ClusterName: fmt.Sprintf("%s-upstream", name),
		}
		configmapMountMode int32 = 420
	)

	if fixture.Context.BinlogConfig.EnableRelayLog {
		drainerConfigModel.RelayPath = "/data/relay"
	}

	drainerConfigmap, _ := RenderDrainerConfig(&drainerConfigModel)
	drainerCommand, _ := RenderDrainerCommand(&drainerCommandModel)

	return &Drainer{
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      drainerName,
			},
			Data: map[string]string{
				"config-file": drainerConfigmap,
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
				ClusterIP: corev1.ClusterIPNone,
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
					"instance": name,
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
								Image:           util.BuildImage("tidb-binlog", fixture.Context.TiDBClusterConfig.ImageVersion, fixture.Context.BinlogConfig.Image),
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
								Env: []corev1.EnvVar{
									{
										Name: "TZ",
										Value: "UTC",
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
												Key:  "config-file",
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
	}
}
