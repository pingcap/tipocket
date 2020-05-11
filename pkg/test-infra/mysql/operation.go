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

package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
)

// Ops knows how to operate MySQL on k8s
type Ops struct {
	cli   client.Client
	mysql *MySQL
}

// New creates a new MySQL Ops.
func New(namespace, name string, conf fixture.MySQLConfig) *Ops {
	return &Ops{
		cli:   tests.TestClient.Cli,
		mysql: newMySQL(namespace, name, conf),
	}
}

// Apply MySQL instance.
func (o *Ops) Apply() error {
	return o.applyMySQL()
}

// Delete MySQL instance.
func (o *Ops) Delete() error {
	if err := o.cli.Delete(context.Background(), o.mysql.Svc); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	if err := o.cli.Delete(context.Background(), o.mysql.Sts); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// GetNodes returns MySQL nodes.
func (o *Ops) GetNodes() ([]clusterTypes.Node, error) {
	pod := &corev1.Pod{} // only 1 replica
	err := o.cli.Get(context.Background(), client.ObjectKey{
		Namespace: o.mysql.Sts.ObjectMeta.Namespace,
		Name:      fmt.Sprintf("%s-0", o.mysql.Sts.ObjectMeta.Name),
	}, pod)
	if err != nil {
		return []clusterTypes.Node{}, err
	}

	return []clusterTypes.Node{{
		Namespace: pod.ObjectMeta.Namespace,
		PodName:   pod.ObjectMeta.Name,
		IP:        pod.Status.PodIP,
		Component: clusterTypes.MySQL,
		Port:      util.FindPort(pod.ObjectMeta.Name, string(clusterTypes.MySQL), pod.Spec.Containers),
	}}, nil
}

// GetClientNodes returns the client nodes.
func (o *Ops) GetClientNodes() ([]clusterTypes.ClientNode, error) {
	var clientNodes []clusterTypes.ClientNode
	ips, err := util.GetNodeIPs(o.cli, o.mysql.Sts.Namespace, o.mysql.Sts.ObjectMeta.Labels)
	if err != nil {
		return clientNodes, err
	} else if len(ips) == 0 {
		return clientNodes, errors.New("k8s node not found")
	}

	svc, err := util.GetServiceByMeta(o.cli, o.mysql.Svc)
	if err != nil {
		return clientNodes, err
	}
	port := getMySQLNodePort(svc)

	for _, ip := range ips {
		clientNodes = append(clientNodes, clusterTypes.ClientNode{
			Namespace:   svc.ObjectMeta.Namespace,
			ClusterName: svc.ObjectMeta.Labels["instance"],
			Component:   clusterTypes.MySQL,
			IP:          ip,
			Port:        port,
		})
	}
	return clientNodes, nil
}

func (o *Ops) applyMySQL() error {
	if err := util.ApplyObject(o.cli, o.mysql.Svc); err != nil {
		return err
	}
	if err := util.ApplyObject(o.cli, o.mysql.Sts); err != nil {
		return err
	}

	if err := o.waitMySQLReady(5 * time.Minute); err != nil {
		return err
	}
	return nil
}

func (o *Ops) waitMySQLReady(timeout time.Duration) error {
	log.Infof("waiting up to %v for StatefulSet %s to be ready", timeout, o.mysql.Sts.Name)
	local := o.mysql.Sts.DeepCopy()
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		key, err := client.ObjectKeyFromObject(local)
		if err != nil {
			return false, err
		}

		if err = o.cli.Get(context.Background(), key, local); err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			log.Errorf("fail to get MySQL StatefulSet %s: %v", local.Name, err)
			return false, err
		}

		if local.Status.ReadyReplicas != *local.Spec.Replicas {
			log.Infof("MySQL %s do not have enough ready replicas, ready: %d, desired: %d",
				local.Name, local.Status.ReadyReplicas, *local.Spec.Replicas)
			return false, nil
		}
		log.Infof("all %d replicas of MySQL %s are ready", local.Status.Replicas, local.Name)
		return true, nil
	})
}

func getMySQLNodePort(svc *corev1.Service) int32 {
	for _, port := range svc.Spec.Ports {
		if port.Port == 3306 {
			return port.NodePort
		}
	}
	return 0
}

// MySQL represents a MySQL instance in K8s.
type MySQL struct {
	Sts *appsv1.StatefulSet
	Svc *corev1.Service
}

// DSN returns a DSN for this MySQL instance.
func (m *MySQL) DSN() string {
	return fmt.Sprintf("root@tcp(%s.%s.svc:3306)/test", m.Svc.Name, m.Svc.Namespace)
}

// newMySQL creates a spec for MySQL.
func newMySQL(namespace, name string, conf fixture.MySQLConfig) *MySQL {
	mysqlName := fmt.Sprintf("tipocket-mysql-%s", name)
	mysqlLabels := map[string]string{
		"app":      "tipocket-mysql",
		"instance": mysqlName,
	}
	version := conf.Version
	if version == "" {
		version = fixture.Context.MySQLVersion
	}

	mysql := &MySQL{
		Svc: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      mysqlName,
				Namespace: namespace,
				Labels:    mysqlLabels,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeNodePort,
				Ports: []corev1.ServicePort{{
					Name:       "mysql",
					Port:       3306,
					TargetPort: intstr.FromInt(3306),
					Protocol:   corev1.ProtocolTCP,
				}},
				Selector: mysqlLabels,
			},
		},
		Sts: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      mysqlName,
				Namespace: namespace,
				Labels:    mysqlLabels,
			},
			Spec: appsv1.StatefulSetSpec{
				ServiceName: mysqlName,
				Replicas:    pointer.Int32Ptr(1),
				Selector:    &metav1.LabelSelector{MatchLabels: mysqlLabels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: mysqlLabels},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{{
							Name:            "remove-lost-and-found",
							Image:           "busybox",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"rm",
								"-rf",
								"/var/lib/mysql/lost+found",
							},
							VolumeMounts: []corev1.VolumeMount{{
								Name:      name,
								MountPath: "/var/lib/mysql",
							}},
						}},
						Containers: []corev1.Container{{
							Name:            "mysql",
							Image:           fmt.Sprintf("mysql:%s", version),
							ImagePullPolicy: corev1.PullIfNotPresent,
							VolumeMounts: []corev1.VolumeMount{{
								Name:      name,
								MountPath: "/var/lib/mysql",
							}},
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ALLOW_EMPTY_PASSWORD",
									Value: "true",
								},
								{
									Name:  "MYSQL_DATABASE",
									Value: "test",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "mysql",
									ContainerPort: 3306,
								},
							},
							Args: []string{"--server-id=1"},
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
						Resources:        fixture.WithStorage(fixture.Small, conf.StorageSize),
					},
				}},
			},
		},
	}

	if conf.EnableBinlog {
		mysql.Sts.Spec.Template.Spec.Containers[0].Args = append(mysql.Sts.Spec.Template.Spec.Containers[0].Args,
			"--log-bin=/var/lib/mysql/mysql-bin")
	}
	if conf.EnableGTID {
		mysql.Sts.Spec.Template.Spec.Containers[0].Args = append(mysql.Sts.Spec.Template.Spec.Containers[0].Args,
			"--enforce-gtid-consistency=ON",
			"--gtid-mode=ON")
	}

	return mysql
}
