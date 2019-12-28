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

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/util"
	"github.com/pingcap/tipocket/test-infra/pkg/fixture"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// MySQLOps knows how to operate MySQL on k8s
type MySQLOps struct {
	cli client.Client
}

func New(cli client.Client) *MySQLOps {
	return &MySQLOps{cli}
}

type MySQLSpec struct {
	Name      string
	Namespace string
	Version   string
	Resource  v1alpha1.Resources
	Storage   fixture.StorageType
}

type MySQL struct {
	Sts *appsv1.StatefulSet
	Svc *corev1.Service
}

func (m *MySQL) URI() string {
	return fmt.Sprintf("root@tcp(%s.%s.svc:3306)/mysql", m.Svc.Name, m.Svc.Namespace)
}

func (m *MySQLOps) ApplyMySQL(spec *MySQLSpec) (*MySQL, error) {
	toCreate, err := m.renderMySQL(spec)
	if err != nil {
		return nil, err
	}
	desiredSts := toCreate.Sts.DeepCopy()
	_, err = controllerutil.CreateOrUpdate(context.TODO(), m.cli, toCreate.Sts, func() error {
		toCreate.Sts.Spec.Template = desiredSts.Spec.Template
		return nil
	})
	desiredSvc := toCreate.Svc.DeepCopy()
	_, err = controllerutil.CreateOrUpdate(context.TODO(), m.cli, toCreate.Svc, func() error {
		clusterIp := toCreate.Svc.Spec.ClusterIP
		toCreate.Svc.Spec = desiredSvc.Spec
		toCreate.Svc.Spec.ClusterIP = clusterIp
		return nil
	})
	return toCreate, nil
}

func (m *MySQLOps) DeleteMySQL(ms *MySQL) error {
	err := m.cli.Delete(context.TODO(), ms.Sts)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	err = m.cli.Delete(context.TODO(), ms.Svc)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (m *MySQLOps) renderMySQL(spec *MySQLSpec) (*MySQL, error) {
	name := fmt.Sprintf("e2e-mysql-%s", spec.Name)
	l := map[string]string{
		"app":      "e2e-mysql",
		"instance": name,
	}
	version := spec.Version
	if version == "" {
		version = fixture.E2eContext.MySQLVersion
	}
	var q resource.Quantity
	var err error
	if spec.Resource.Requests != nil {
		size := spec.Resource.Requests.Storage
		q, err = resource.ParseQuantity(size)
		if err != nil {
			return nil, fmt.Errorf("cant' get storage size for mysql: %v", err)
		}
	}
	return &MySQL{
		Sts: &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: spec.Namespace,
				Labels:    l,
			},
			Spec: appsv1.StatefulSetSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{MatchLabels: l},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: l},
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
								Name:      spec.Name,
								MountPath: "/var/lib/mysql",
							}},
						}},
						Containers: []corev1.Container{{
							Name:            "mysql",
							Image:           fmt.Sprintf("mysql:%s", version),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources:       util.ResourceRequirement(spec.Resource),
							VolumeMounts: []corev1.VolumeMount{{
								Name:      spec.Name,
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
						}},
					},
				},
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{{
					ObjectMeta: metav1.ObjectMeta{Name: spec.Name},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce,
						},
						StorageClassName: pointer.StringPtr(fixture.StorageClass(spec.Storage)),
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: q,
							},
						},
					},
				}},
			},
		},
		Svc: &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: spec.Namespace,
				Labels:    l,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Ports: []corev1.ServicePort{{
					Name:       "mysql",
					Port:       3306,
					TargetPort: intstr.FromInt(3306),
					Protocol:   corev1.ProtocolTCP,
				}},
				Selector: l,
			},
		},
	}, nil
}
