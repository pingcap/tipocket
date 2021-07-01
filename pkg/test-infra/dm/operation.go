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

package dm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ngaut/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
)

// Ops knows how to operate DM cluster on K8s.
type Ops struct {
	cli client.Client
	dm  *DM
}

// New creates a new DM Ops.
func New(namespace, name string, conf fixture.DMConfig) *Ops {
	return &Ops{
		cli: tests.TestClient.Cli,
		dm:  newDM(namespace, name, conf),
	}
}

// Apply DM cluster.
func (o *Ops) Apply() error {
	return o.applyDM()
}

// Delete DM cluster.
func (o *Ops) Delete() error {
	objs := []runtime.Object{o.dm.SvcWorker, o.dm.StsWorker, o.dm.SvcMaster, o.dm.StsMaster}
	for _, obj := range objs {
		if err := o.cli.Delete(context.Background(), obj); err != nil {
			return err
		}
	}
	return nil
}

// GetNodes returns DM (DM-master & DM-worker) nodes.
func (o *Ops) GetNodes() ([]cluster.Node, error) {
	podsMaster := &corev1.PodList{}
	if err := o.cli.List(context.Background(), podsMaster,
		client.InNamespace(o.dm.StsMaster.ObjectMeta.Namespace),
		client.MatchingLabels(o.dm.StsMaster.ObjectMeta.Labels)); err != nil {
		return []cluster.Node{}, err
	}

	podsWorker := &corev1.PodList{}
	if err := o.cli.List(context.Background(), podsWorker,
		client.InNamespace(o.dm.StsMaster.ObjectMeta.Namespace),
		client.MatchingLabels(o.dm.StsWorker.ObjectMeta.Labels)); err != nil {
		return []cluster.Node{}, err
	}

	nodes := make([]cluster.Node, 0, len(podsMaster.Items)+len(podsWorker.Items))
	// because dmMasters have a NodePort kind service, dmWorkers have a headless kind service,
	// and they all are managed by statefulset,
	// so we use their fqdn as node ip here.
	for _, pod := range podsMaster.Items {
		fqdn, err := util.GetFQDNFromStsPod(&pod)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, cluster.Node{
			Namespace: pod.ObjectMeta.Namespace,
			PodName:   pod.ObjectMeta.Name,
			IP:        fqdn,
			Component: cluster.DM,
			Port:      util.FindPort(pod.ObjectMeta.Name, string(cluster.DM), pod.Spec.Containers),
		})
	}
	for _, pod := range podsWorker.Items {
		nodes = append(nodes, cluster.Node{
			Namespace: pod.ObjectMeta.Namespace,
			PodName:   pod.ObjectMeta.Name,
			IP:        pod.Status.PodIP,
			Component: cluster.DM,
			Port:      util.FindPort(pod.ObjectMeta.Name, string(cluster.DM), pod.Spec.Containers),
		})
	}

	return nodes, nil
}

// GetClientNodes returns the client nodes.
func (o *Ops) GetClientNodes() ([]cluster.ClientNode, error) {
	var clientNodes []cluster.ClientNode
	ips, err := util.GetNodeIPsFromPod(o.cli, o.dm.StsMaster.Namespace, o.dm.StsMaster.ObjectMeta.Labels)
	if err != nil {
		return clientNodes, err
	} else if len(ips) == 0 {
		return clientNodes, errors.New("k8s node not found")
	}

	svc, err := util.GetServiceByMeta(o.cli, o.dm.SvcMaster)
	if err != nil {
		return clientNodes, err
	}
	port := getDMMasterNodePort(svc)

	for _, ip := range ips {
		clientNodes = append(clientNodes, cluster.ClientNode{
			Namespace:   svc.ObjectMeta.Namespace,
			ClusterName: svc.ObjectMeta.Labels["instance"],
			Component:   cluster.DM,
			IP:          ip,
			Port:        port,
		})
	}

	return clientNodes, nil
}

func (o *Ops) applyDM() error {
	if err := util.ApplyObject(o.cli, o.dm.SvcMaster); err != nil {
		return err
	}
	if err := util.ApplyObject(o.cli, o.dm.SvcWorker); err != nil {
		return err
	}

	if err := util.ApplyObject(o.cli, o.dm.StsMaster); err != nil {
		return err
	}
	if err := o.waitStsReady(5*time.Minute, o.dm.StsMaster); err != nil {
		return err
	}

	if err := util.ApplyObject(o.cli, o.dm.StsWorker); err != nil {
		return err
	}
	if err := o.waitStsReady(5*time.Minute, o.dm.StsWorker); err != nil {
		return err
	}

	return nil
}

func (o *Ops) waitStsReady(timeout time.Duration, sts *appsv1.StatefulSet) error {
	log.Infof("waiting up to %v for StatefulSet %s to be ready", timeout, sts.Name)
	local := sts.DeepCopy()
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		key, err := client.ObjectKeyFromObject(local)
		if err != nil {
			return false, err
		}

		if err = o.cli.Get(context.Background(), key, local); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			log.Errorf("fail to get DM StatefulSet %s: %v", local.Name, err)
			return false, err
		}

		if local.Status.ReadyReplicas != *local.Spec.Replicas {
			log.Infof("DM %s do not have enough ready replicas, ready: %d, desired: %d",
				local.Name, local.Status.ReadyReplicas, *local.Spec.Replicas)
			return false, nil
		}
		log.Infof("all %d replicas of DM %s are ready", local.Status.Replicas, local.Name)
		return true, nil
	})
}

func getDMMasterNodePort(svc *corev1.Service) int32 {
	for _, port := range svc.Spec.Ports {
		if port.Port == 8261 {
			return port.NodePort
		}
	}
	return 0
}

// DM represents a DM cluster in K8s.
type DM struct {
	SvcMaster *corev1.Service
	SvcWorker *corev1.Service
	StsMaster *appsv1.StatefulSet
	StsWorker *appsv1.StatefulSet
}

// newDM creates a spec for DM.
func newDM(namespace, name string, conf fixture.DMConfig) *DM {
	dmMasterName := fmt.Sprintf("tipocket-dm-master-%s", name)
	dmMasterLabels := map[string]string{
		"app":      "tipocket-dm",
		"instance": dmMasterName,
	}

	dmWorkerName := fmt.Sprintf("tipocket-dm-worker-%s", name)
	dmWorkerLabels := map[string]string{
		"app":      "tipocket-dm",
		"instance": dmWorkerName,
	}

	image := fmt.Sprintf("pingcap/dm:%s", conf.DMVersion)

	dmMasterInitCluster := "--initial-cluster="
	for i := 0; i < conf.MasterReplica; i++ {
		dmMasterInitCluster += fmt.Sprintf("%[1]s-%[3]d=http://%[1]s-%[3]d.%[1]s.%[2]s:8291", dmMasterName, namespace, i)
		if i+1 != conf.MasterReplica {
			dmMasterInitCluster += ","
		}
	}

	dm := &DM{
		SvcMaster: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dmMasterName,
				Namespace: namespace,
				Labels:    dmMasterLabels,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
				Ports: []corev1.ServicePort{{
					Name:       "dm-master",
					Port:       8261,
					TargetPort: intstr.FromInt(8261),
					Protocol:   corev1.ProtocolTCP,
				}, {
					Name:       "dm-master-peer",
					Port:       8291,
					TargetPort: intstr.FromInt(8291),
					Protocol:   corev1.ProtocolTCP,
				}},
				Selector: dmMasterLabels,
			},
		},
		SvcWorker: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dmWorkerName,
				Namespace: namespace,
				Labels:    dmWorkerLabels,
			},
			Spec: corev1.ServiceSpec{
				Type:      corev1.ServiceTypeClusterIP,
				ClusterIP: corev1.ClusterIPNone,
				Ports: []corev1.ServicePort{{
					Name:       "dm-worker",
					Port:       8262,
					TargetPort: intstr.FromInt(8262),
					Protocol:   corev1.ProtocolTCP,
				}},
				Selector: dmWorkerLabels,
			},
		},
		StsMaster: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dmMasterName,
				Namespace: namespace,
				Labels:    dmMasterLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName:         dmMasterName,
				PodManagementPolicy: appsv1.ParallelPodManagement,
				Replicas:            pointer.Int32Ptr(int32(conf.MasterReplica)),
				Selector:            &metav1.LabelSelector{MatchLabels: dmMasterLabels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: dmMasterLabels,
						Annotations: map[string]string{
							"prometheus.io/path":   "/metrics",
							"prometheus.io/port":   "8261",
							"prometheus.io/scrape": "true",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:            "dm",
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      name,
									MountPath: "/data",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "dm-master",
									ContainerPort: 8261,
								},
								{
									Name:          "dm-master-peer",
									ContainerPort: 8291,
								},
							},
							Command: []string{
								"/dm-master",
								"--data-dir=/data",
								"--name=$(MY_POD_NAME)",
								"--master-addr=:8261",
								fmt.Sprintf("--advertise-addr=$(MY_POD_NAME).%s.%s:8261", dmMasterName, namespace),
								"--peer-urls=:8291",
								fmt.Sprintf("--advertise-peer-urls=http://$(MY_POD_NAME).%s.%s:8291", dmMasterName, namespace),
								dmMasterInitCluster,
							},
							Env: []corev1.EnvVar{
								{
									Name: "MY_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.name",
										},
									},
								},
							},
						}},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{Name: name},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: &fixture.Context.LocalVolumeStorageClass,
						Resources:        fixture.WithStorage(fixture.Small, "1Gi"),
					},
				}},
			},
		},
		StsWorker: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dmWorkerName,
				Namespace: namespace,
				Labels:    dmWorkerLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName:         dmWorkerName,
				PodManagementPolicy: appsv1.ParallelPodManagement,
				Replicas:            pointer.Int32Ptr(int32(conf.WorkerReplica)),
				Selector:            &metav1.LabelSelector{MatchLabels: dmWorkerLabels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: dmWorkerLabels,
						Annotations: map[string]string{
							"prometheus.io/path":   "/metrics",
							"prometheus.io/port":   "8262",
							"prometheus.io/scrape": "true",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name:            "dm",
							Image:           image,
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      name,
									MountPath: "/data",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "client",
									ContainerPort: 8262,
								},
							},
							Command: []string{
								"/dm-worker",
								"--name=$(MY_POD_NAME)",
								"--worker-addr=:8262",
								fmt.Sprintf("--advertise-addr=$(MY_POD_NAME).%s.%s:8262", dmWorkerName, namespace),
								fmt.Sprintf("--join=%s.%s:8261", dmMasterName, namespace),
							},
							Env: []corev1.EnvVar{
								{
									Name: "MY_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.name",
										},
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								// NOTE: no `ReadinessProbe` for DM-master now, because
								// - DM-master relies on endpoint to communicate with other members when starting
								// - but endpoint may no available if DM-master not started
								// then DM-master may never become ready.
								// so we only check the readiness for DM-worker.
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/status",
										Port:   intstr.FromInt(8262),
										Scheme: "HTTP",
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       5,
								FailureThreshold:    5,
							},
						}},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{Name: name},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: &fixture.Context.LocalVolumeStorageClass,
						Resources:        fixture.WithStorage(fixture.Small, "1Gi"),
					},
				}},
			},
		},
	}

	return dm
}
