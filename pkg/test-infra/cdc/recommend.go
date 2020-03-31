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

package cdc

import (
	"fmt"
	"strings"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// CDC defines the configuration for running on Kubernetes
type CDC struct {
	*appsv1.StatefulSet
	*corev1.Service
	*batchv1.Job
}

// RecommendedCDCCluster creates cluster with CDC
func newCDC(ns, name string) *CDC {
	var (
		cdcName        = fmt.Sprintf("%s-cdc", name)
		cdcJobName     = fmt.Sprintf("%s-job", cdcName)
		upstreamPDAddr = fmt.Sprintf("%s-upstream-pd", name)
		downstreamDB   = fmt.Sprintf("%s-downstream-tidb", name)
		cdcLabels      = map[string]string{
			"app.kubernetes.io/name":      "tidb-cluster",
			"app.kubernetes.io/component": "cdc",
			"app.kubernetes.io/instance":  cdcName,
		}
	)

	return &CDC{
		Service: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cdcName,
				Namespace: ns,
				Labels:    cdcLabels,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{
						Name: "http",
						Port: 8300,
					},
				},
				ClusterIP: corev1.ClusterIPNone,
				Selector:  cdcLabels,
			},
		},
		StatefulSet: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cdcName,
				Namespace: ns,
				Labels:    cdcLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: cdcName,
				Replicas:    pointer.Int32Ptr(3),
				Selector:    &metav1.LabelSelector{MatchLabels: cdcLabels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: cdcLabels},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name: "cdc",
							Command: []string{
								"/cdc",
								"server",
								fmt.Sprintf("--pd=%s", fmt.Sprintf("http://%s:2379", upstreamPDAddr)),
								"--status-addr=0.0.0.0:8300",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8300,
								},
							},
							Image:           buildCDCImage("ticdc"),
							ImagePullPolicy: corev1.PullAlways,
						}},
					},
				},
			},
		},
		Job: &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cdcJobName,
				Namespace: ns,
				Labels: map[string]string{
					"app.kubernetes.io/name":      "tidb-cluster",
					"app.kubernetes.io/component": "cdc",
					"app.kubernetes.io/instance":  cdcJobName,
				},
			},
			Spec: batchv1.JobSpec{
				TTLSecondsAfterFinished: pointer.Int32Ptr(10),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "cdc-cli",
								Image:           buildCDCImage("ticdc"),
								ImagePullPolicy: corev1.PullAlways,
								Command: []string{
									"/cdc",
									"cli",
									"changefeed",
									"create",
									fmt.Sprintf("--pd=%s", fmt.Sprintf("http://%s:2379", upstreamPDAddr)),
									fmt.Sprintf("--sink-uri=mysql://root@%s:4000/", downstreamDB),
									"--start-ts=0",
								},
							},
						},
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
			},
		},
	}
}

func buildCDCImage(name string) string {
	var (
		b                strings.Builder
		version          = fixture.Context.ImageVersion
		dockerRepository = fixture.Context.CDCConfig.DockerRepository
		hubAddress       = fixture.Context.CDCConfig.HubAddress
	)

	if fixture.Context.CDCConfig.CDCVersion != "" {
		version = fixture.Context.CDCConfig.CDCVersion
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
