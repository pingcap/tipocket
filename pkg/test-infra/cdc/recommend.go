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

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
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
		sinkURI  string
		logLevel = fixture.Context.CDCConfig.LogLevel
		timezone = fixture.Context.CDCConfig.Timezone
	)
	if fixture.Context.CDCConfig.EnableKafka {
		kafkaName := fmt.Sprintf("%s-kafka", name)
		sinkURI = fmt.Sprintf("kafka://%s:9092/cdc-test?partition-num=6&max-message-bytes=67108864&replication-factor=1", kafkaName)
	} else {
		sinkURI = fmt.Sprintf("mysql://root@%s:4000/", downstreamDB)
	}
	logVol := corev1.VolumeMount{
		Name:      "log",
		MountPath: "/var/log/cdc",
	}

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
						Containers: []corev1.Container{
							{
								Name: "cdc",
								Command: []string{
									"/cdc",
									"server",
									fmt.Sprintf("--pd=%s", fmt.Sprintf("http://%s:2379", upstreamPDAddr)),
									"--addr=0.0.0.0:8300",
									fmt.Sprintf("--advertise-addr=%s", fmt.Sprintf("http://$(POD_NAME).%s.%s:8300", cdcName, ns)),
									"--log-file", "/var/log/cdc/cdc.log",
									"--log-level", logLevel,
									"--tz", timezone,
								},
								Env: []corev1.EnvVar{
									{
										Name: "POD_NAME",
										ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												APIVersion: "v1",
												FieldPath:  "metadata.name",
											},
										},
									},
								},
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 8300,
									},
								},
								Image:           util.BuildImage("ticdc", fixture.Context.TiDBClusterConfig.ImageVersion, fixture.Context.CDCConfig.Image),
								ImagePullPolicy: corev1.PullAlways,
								VolumeMounts:    []corev1.VolumeMount{logVol},
							},
							{
								Name: "cdc-log",
								Command: []string{
									"/bin/sh",
									"-c",
									`touch /var/log/cdc/cdc.log; tail -n0 -F /var/log/cdc/cdc.log`,
								},
								Image:           "busybox",
								ImagePullPolicy: corev1.PullIfNotPresent,
								VolumeMounts:    []corev1.VolumeMount{logVol},
							},
						},
						Volumes: []corev1.Volume{{
							Name: "log",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
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
								Image:           util.BuildImage("ticdc", fixture.Context.TiDBClusterConfig.ImageVersion, fixture.Context.CDCConfig.Image),
								ImagePullPolicy: corev1.PullAlways,
								Command: []string{
									"/cdc",
									"cli",
									"changefeed",
									"create",
									fmt.Sprintf("--pd=%s", fmt.Sprintf("http://%s:2379", upstreamPDAddr)),
									fmt.Sprintf("--sink-uri=%s", sinkURI),
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

// Zookeeper ...
type Zookeeper struct {
	*appsv1.StatefulSet
	*corev1.Service
}

func newZookeeper(ns, name string) *Zookeeper {
	var (
		zkName   = fmt.Sprintf("%s-zookeeper", name)
		zkLabels = map[string]string{
			"app.kubernetes.io/name":      "tidb-cluster",
			"app.kubernetes.io/component": "zookeeper",
			"app.kubernetes.io/instance":  zkName,
		}
	)
	logVol := corev1.VolumeMount{
		Name:      "log",
		MountPath: "/var/log/zookeeper",
	}

	return &Zookeeper{
		Service: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      zkName,
				Namespace: ns,
				Labels:    zkLabels,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{
						Name: "http",
						Port: 2181,
					},
				},
				ClusterIP: corev1.ClusterIPNone,
				Selector:  zkLabels,
			},
		},
		StatefulSet: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      zkName,
				Namespace: ns,
				Labels:    zkLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: zkName,
				Replicas:    pointer.Int32Ptr(1),
				Selector:    &metav1.LabelSelector{MatchLabels: zkLabels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: zkLabels},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "zookeeper",
								Image:           "zookeeper",
								ImagePullPolicy: corev1.PullIfNotPresent,
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 2181,
									},
								},
								VolumeMounts: []corev1.VolumeMount{logVol},
							},
						},
						Volumes: []corev1.Volume{{
							Name: "log",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						}},
					},
				},
			},
		},
	}
}

// Kafka ...
type Kafka struct {
	*appsv1.StatefulSet
	*corev1.Service
	*batchv1.Job
	*Zookeeper
}

func newKafka(ns, name string) *Kafka {
	var (
		kafkaName         = fmt.Sprintf("%s-kafka", name)
		kafkaConsumerName = fmt.Sprintf("%s-consumer", kafkaName)
		kafkaLabels       = map[string]string{
			"app.kubernetes.io/name":      "tidb-cluster",
			"app.kubernetes.io/component": "kafka",
			"app.kubernetes.io/instance":  kafkaName,
		}
		upstreamURI   = fmt.Sprintf("kafka://%s:9092/cdc-test?partition-num=6&max-message-bytes=67108864&replication-factor=1", kafkaName)
		downstreamURI = fmt.Sprintf("mysql://root@%s-downstream-tidb:4000/", name)
		zkName        = fmt.Sprintf("%s-zookeeper", name)
	)
	logVol := corev1.VolumeMount{
		Name:      "log",
		MountPath: "/var/log/kafka",
	}

	return &Kafka{
		Service: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kafkaName,
				Namespace: ns,
				Labels:    kafkaLabels,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{
					{
						Name: "http",
						Port: 9092,
					},
				},
				ClusterIP: corev1.ClusterIPNone,
				Selector:  kafkaLabels,
			},
		},
		StatefulSet: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kafkaName,
				Namespace: ns,
				Labels:    kafkaLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: kafkaName,
				Replicas:    pointer.Int32Ptr(1),
				Selector:    &metav1.LabelSelector{MatchLabels: kafkaLabels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: kafkaLabels},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "kafka",
								Env: []corev1.EnvVar{
									{
										Name:  "KAFKA_MESSAGE_MAX_BYTES",
										Value: "1073741824",
									},
									{
										Name:  "KAFKA_REPLICA_FETCH_MAX_BYTES",
										Value: "1073741824",
									},
									{
										Name:  "KAFKA_ADVERTISED_PORT",
										Value: "9092",
									},
									{
										Name:  "KAFKA_ADVERTISED_HOST_NAME",
										Value: kafkaName,
									},
									{
										Name:  "KAFKA_BROKER_ID",
										Value: "1",
									},
									{
										Name:  "KAFKA_ZOOKEEPER_CONNECT",
										Value: fmt.Sprintf("%s:2181", zkName),
									},
								},
								Image:           "wurstmeister/kafka",
								ImagePullPolicy: corev1.PullIfNotPresent,
								Ports: []corev1.ContainerPort{
									{
										Name:          "http",
										ContainerPort: 9092,
									},
								},
								VolumeMounts: []corev1.VolumeMount{logVol},
							},
						},
						Volumes: []corev1.Volume{{
							Name: "log",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						}},
					},
				},
			},
		},
		Job: &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kafkaConsumerName,
				Namespace: ns,
				Labels: map[string]string{
					"app.kubernetes.io/name":      "tidb-cluster",
					"app.kubernetes.io/component": "kafka-consumer",
					"app.kubernetes.io/instance":  kafkaConsumerName,
				},
			},
			Spec: batchv1.JobSpec{
				TTLSecondsAfterFinished: pointer.Int32Ptr(10),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "consumer",
								Image:           fixture.Context.CDCConfig.KafkaConsumerImage,
								ImagePullPolicy: corev1.PullAlways,
								Command: []string{
									"/cdc_kafka_consumer",
									"--upstream-uri", upstreamURI,
									"--downstream-uri", downstreamURI,
									"--log-file", "/var/log/kafka/consumer.log",
									"--log-level", "debug",
								},
							},
						},
						RestartPolicy: corev1.RestartPolicyOnFailure,
					},
				},
			},
		},
		Zookeeper: newZookeeper(ns, name),
	}
}
