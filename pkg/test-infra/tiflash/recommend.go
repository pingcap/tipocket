// Copyright 2020 PingCAP, Inc.
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

package tiflash

import (
	"fmt"
	"strings"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	configTmplName = "config_templ.toml"
	userTmplName   = "users.toml"
	proxyTmplName  = "proxy_templ.toml"

	busyboxImage = "busybox"
)

// CDC defines the configuration for running on Kubernetes
type TiFlash struct {
	*appsv1.StatefulSet
	*corev1.ConfigMap
	*corev1.Service
}

// Recommendation defines binlog cluster
type Recommendation struct {
	*TiFlash
	TiDBCluster *tidb.Recommendation
	NS          string
	Name        string
}

// RecommendedTiFlashCluster creates cluster with TiFlash
func RecommendedTiFlashCluster(ns, name, version string) *Recommendation {
	var (
		tiFlashName = name + "-tiflash"
		lbls        = map[string]string{
			"app.kubernetes.io/name":      "tidb-cluster",
			"app.kubernetes.io/component": "tiflash",
			"app.kubernetes.io/instance":  tiFlashName,
		}
		model = &tiFlashConfig{ClusterName: name, Namespace: ns}
	)

	return &Recommendation{
		NS:          ns,
		Name:        name,
		TiDBCluster: tidb.RecommendedTiDBCluster(ns, fmt.Sprintf("%s", name), version),
		TiFlash: &TiFlash{
			StatefulSet: tiFlashStatefulSet(tiFlashName, lbls, model),
			// we use name instead of tiFlashName here
			// because we want to use it to do template rendering.
			ConfigMap: tiFlashConfigMap(name, model),
			Service:   tiFlashService(ns, tiFlashName, lbls),
		},
	}
}

func tiFlashStatefulSet(name string, lbls map[string]string, model *tiFlashConfig) *appsv1.StatefulSet {
	dataVol := corev1.VolumeMount{
		Name:      "tiflash",
		MountPath: "/data",
	}
	configVol := corev1.VolumeMount{
		Name:      "config",
		MountPath: "/etc/tiflash",
	}

	cmd, _ := renderTiFlashCmd(model)
	image := buildTiFlashImage("tics")

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: model.Namespace,
			Labels:    lbls,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name,
			Replicas:    pointer.Int32Ptr(1),
			Selector:    &metav1.LabelSelector{MatchLabels: lbls},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: lbls},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:            "init-tiflash",
							Image:           image,
							Command:         []string{"bash", "-c", tiFlashInitCmdTemplate},
							VolumeMounts:    []corev1.VolumeMount{dataVol, configVol},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "tiflash",
							Command:         []string{"bash", "-c", cmd},
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env:             []corev1.EnvVar{{Name: "LD_LIBRARY_PATH", Value: "/"}},
							VolumeMounts:    []corev1.VolumeMount{dataVol, configVol},
							Ports: []corev1.ContainerPort{
								{
									Name:          "flash",
									ContainerPort: 3930,
								},
								{
									Name:          "http",
									ContainerPort: 8123,
								},
								{
									Name:          "metrics",
									ContainerPort: 8234,
								},
								{
									Name:          "tcp",
									ContainerPort: 9000,
								},
							},
						},
						{
							Name: "tiflash-log",
							Command: []string{
								"/bin/sh",
								"-c",
								`touch /data/logs/server.log; tail -n0 -F /data/logs/server.log`,
							},
							Image:           busyboxImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts:    []corev1.VolumeMount{dataVol},
						},
						{
							Name: "error-log",
							Command: []string{
								"/bin/sh",
								"-c",
								`touch /data/logs/error.log; tail -n0 -F /data/logs/error.log`,
							},
							Image:           busyboxImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts:    []corev1.VolumeMount{dataVol},
						},
					},
					Volumes: []corev1.Volume{{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: name,
								},
								Items: []corev1.KeyToPath{
									{Key: configTmplName, Path: configTmplName},
									{Key: userTmplName, Path: userTmplName},
									{Key: proxyTmplName, Path: proxyTmplName},
								},
							},
						},
					}},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "tiflash",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &fixture.Context.LocalVolumeStorageClass,
						Resources:        fixture.WithStorage(fixture.Medium, "50Gi"),
					},
				},
			},
		},
	}
}

func tiFlashConfigMap(name string, model *tiFlashConfig) *corev1.ConfigMap {
	conf, _ := renderTiFlashConfig(model)
	proxyConf, _ := renderTiFlashProxyTpl(model)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: model.Namespace,
			Name:      name + "-tiflash",
		},

		Data: map[string]string{
			configTmplName: conf,
			userTmplName:   usersTemplate,
			proxyTmplName:  proxyConf,
		},
	}
}

func tiFlashService(ns, name string, lbls map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels:    lbls,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Name: "flash",
					Port: 3930,
				},
				{
					Name: "http",
					Port: 8123,
				},
				{
					Name: "metrics",
					Port: 8234,
				},
				{
					Name: "tcp",
					Port: 9000,
				},
			},
			ClusterIP: corev1.ClusterIPNone,
			Selector:  lbls,
		},
	}
}

func buildTiFlashImage(name string) string {
	var (
		b                strings.Builder
		version          = fixture.Context.ImageVersion
		dockerRepository = fixture.Context.TiFlashConfig.DockerRepository
		hubAddress       = fixture.Context.TiFlashConfig.HubAddress
	)

	if fixture.Context.TiFlashConfig.TiFlashVersion != "" {
		version = fixture.Context.TiFlashConfig.TiFlashVersion
	}

	if hubAddress == "" {
		hubAddress = fixture.Context.HubAddress
	}

	if hubAddress != "" {
		fmt.Fprintf(&b, "%s/", hubAddress)
	}

	if dockerRepository == "" {
		dockerRepository = fixture.Context.DockerRepository
	}

	b.WriteString(dockerRepository)
	b.WriteString("/")
	b.WriteString(name)
	b.WriteString(":")
	b.WriteString(version)
	return b.String()
}
