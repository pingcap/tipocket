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
	"context"
	"fmt"
	"time"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// TidbOps knows how to operate TiDB on k8s
type TidbOps struct {
	cli client.Client
}

func New(cli client.Client) *TidbOps {
	return &TidbOps{cli}
}

func (t *TidbOps) GetTiDBService(tc *v1alpha1.TidbCluster) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-tidb", tc.Name),
			Namespace: tc.Namespace,
		},
	}
	key, err := client.ObjectKeyFromObject(svc)
	if err != nil {
		return nil, err
	}

	if err := t.cli.Get(context.TODO(), key, svc); err != nil {
		return nil, err
	}

	return svc, nil
}

func (t *TidbOps) GetTiDBServiceByMeta(meta *metav1.ObjectMeta) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: *meta,
	}
	key, err := client.ObjectKeyFromObject(svc)
	if err != nil {
		return nil, err
	}

	if err := t.cli.Get(context.TODO(), key, svc); err != nil {
		return nil, err
	}

	return svc, nil
}

func (t *TidbOps) GetTiDBNodePort(tc *v1alpha1.TidbCluster) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-tidb", tc.Name),
			Namespace: tc.Namespace,
		},
	}
	key, err := client.ObjectKeyFromObject(svc)
	if err != nil {
		return nil, err
	}

	if err := t.cli.Get(context.TODO(), key, svc); err != nil {
		return nil, err
	}

	return svc, nil
}

func (t *TidbOps) GetNodes(ns string) (*corev1.PodList, error) {
	pods := &corev1.PodList{}
	err := t.cli.List(context.TODO(), pods)
	if err != nil {
		return nil, err
	}
	// filter namespace
	r := pods.DeepCopy()
	r.Items = []corev1.Pod{}
	for _, pod := range pods.Items {
		if pod.ObjectMeta.Namespace == ns {
			r.Items = append(r.Items, pod)
		}
	}
	return r, nil
}

func (t *TidbOps) ApplyTiDBCluster(tc *v1alpha1.TidbCluster) error {
	desired := tc.DeepCopy()
	if tc.Spec.Version == "" {
		tc.Spec.Version = fixture.E2eContext.TiDBVersion
	}

	klog.Info("Apply tidb discovery")
	if err := t.ApplyDiscovery(tc); err != nil {
		return err
	}
	klog.Info("Apply tidb configmap")
	if err := t.ApplyTiDBConfigMap(tc); err != nil {
		return err
	}
	klog.Info("Apply pd configmap")
	if err := t.ApplyPDConfigMap(tc); err != nil {
		return err
	}
	klog.Info("Apply tikv configmap")
	if err := t.ApplyTiKVConfigMap(tc); err != nil {
		return err
	}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), t.cli, tc, func() error {
		tc.Spec = desired.Spec
		tc.Annotations = desired.Annotations
		tc.Labels = desired.Labels
		return nil
	})
	return err
}

func (t *TidbOps) WaitTiDBClusterReady(tc *v1alpha1.TidbCluster, timeout time.Duration) error {
	local := tc.DeepCopy()
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		key, err := client.ObjectKeyFromObject(local)
		if err != nil {
			return false, err
		}
		err = t.cli.Get(context.TODO(), key, local)
		if err != nil && errors.IsNotFound(err) {
			return false, err
		}
		if err != nil {
			klog.Warningf("error getting tidbcluster: %v", err)
			return false, nil
		}
		if local.Status.PD.StatefulSet == nil {
			return false, nil
		}
		pdReady, pdDesired := local.Status.PD.StatefulSet.ReadyReplicas, local.Spec.PD.Replicas
		if pdReady < pdDesired {
			klog.V(4).Infof("PD do not have enough ready replicas, ready: %d, desired: %d", pdReady, pdDesired)
			return false, nil
		}
		if local.Status.TiKV.StatefulSet == nil {
			return false, nil
		}
		tikvReady, tikvDesired := local.Status.TiKV.StatefulSet.ReadyReplicas, local.Spec.TiKV.Replicas
		if tikvReady < tikvDesired {
			klog.V(4).Infof("TiKV do not have enough ready replicas, ready: %d, desired: %d", tikvReady, tikvDesired)
			return false, nil
		}
		if local.Status.TiDB.StatefulSet == nil {
			return false, nil
		}
		tidbReady, tidbDesired := local.Status.TiDB.StatefulSet.ReadyReplicas, local.Spec.TiDB.Replicas
		if tidbReady < tidbDesired {
			klog.V(4).Infof("TiDB do not have enough ready replicas, ready: %d, desired: %d", tidbReady, tidbDesired)
			return false, nil
		}
		return true, nil
	})
}

func (t *TidbOps) DeleteTiDBCluster(tc *v1alpha1.TidbCluster) error {
	return t.cli.Delete(context.TODO(), tc)
}

func (t *TidbOps) ApplyTiDBConfigMap(tc *v1alpha1.TidbCluster) error {
	configMap, err := getTiDBConfigMap(tc)
	if err != nil {
		return err
	}
	err = t.cli.Create(context.TODO(), configMap)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (t *TidbOps) ApplyPDConfigMap(tc *v1alpha1.TidbCluster) error {
	configMap, err := getPDConfigMap(tc)
	if err != nil {
		return err
	}
	err = t.cli.Create(context.TODO(), configMap)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (t *TidbOps) ApplyTiKVConfigMap(tc *v1alpha1.TidbCluster) error {
	configMap, err := getTiKVConfigMap(tc)
	if err != nil {
		return err
	}
	err = t.cli.Create(context.TODO(), configMap)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (t *TidbOps) ApplyTiDBService(s *corev1.Service) error {
	err := t.cli.Create(context.TODO(), s)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (t *TidbOps) ApplyDiscovery(tc *v1alpha1.TidbCluster) error {

	meta, _ := getDiscoveryMeta(tc)

	// Ensure RBAC
	err := t.cli.Create(context.TODO(), &rbacv1.Role{
		ObjectMeta: meta,
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{"pingcap.com"},
				Resources:     []string{"tidbclusters"},
				ResourceNames: []string{tc.Name},
				Verbs:         []string{"get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			},
		},
	})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	err = t.cli.Create(context.TODO(), &corev1.ServiceAccount{
		ObjectMeta: meta,
	})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	err = t.cli.Create(context.TODO(), &rbacv1.RoleBinding{
		ObjectMeta: meta,
		Subjects: []rbacv1.Subject{{
			Kind: "ServiceAccount",
			Name: meta.Name,
		}},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     meta.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
	})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// RBAC ensured, reconcile
	err = t.cli.Create(context.TODO(), getTidbDiscoveryService(tc))
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	err = t.cli.Create(context.TODO(), getTidbDiscoveryDeployment(tc))
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (t *TidbOps) GetPDMember(namespace, name string) (string, []string, error) {
	var local v1alpha1.TidbCluster
	var members []string
	err := wait.PollImmediate(5*time.Second, time.Minute*time.Duration(5), func() (bool, error) {
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		err := t.cli.Get(context.TODO(), key, &local)
		if err != nil && errors.IsNotFound(err) {
			return false, err
		}
		if err != nil {
			klog.Warningf("error getting tidbcluster: %v", err)
			return false, nil
		}
		if local.Status.PD.StatefulSet == nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", nil, err
	}
	for _, member := range local.Status.PD.Members {
		members = append(members, member.Name)
	}
	return local.Status.PD.Leader.Name, members, nil
}

func getTidbDiscoveryService(tc *v1alpha1.TidbCluster) *corev1.Service {
	meta, l := getDiscoveryMeta(tc)
	return &corev1.Service{
		ObjectMeta: meta,
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{
				Name:       "discovery",
				Port:       10261,
				TargetPort: intstr.FromInt(10261),
				Protocol:   corev1.ProtocolTCP,
			}},
			Selector: l.Labels(),
		},
	}
}

// TODO: use the latest tidb-operator feature instead
func getPDConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	s, err := RenderPDStartScript(&PDStartScriptModel{})
	if err != nil {
		return nil, err
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tc.Namespace,
			Name:      fmt.Sprintf("%s-pd", tc.Name),
		},
		Data: map[string]string{
			"startup-script": s,
			"config-file":    ``,
		},
	}, nil
}

func getTiDBConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	s, err := RenderTiDBStartScript(&TidbStartScriptModel{ClusterName: tc.Name})
	if err != nil {
		return nil, err
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tc.Namespace,
			Name:      fmt.Sprintf("%s-tidb", tc.Name),
		},
		Data: map[string]string{
			"startup-script": s,
			"config-file":    ``,
		},
	}, nil
}

func getTiKVConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	s, err := RenderTiKVStartScript(&TiKVStartScriptModel{})
	if err != nil {
		return nil, err
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tc.Namespace,
			Name:      fmt.Sprintf("%s-tikv", tc.Name),
		},
		Data: map[string]string{
			"startup-script": s,
			"config-file":    ``,
		},
	}, nil
}

func getTidbDiscoveryDeployment(tc *v1alpha1.TidbCluster) *appsv1.Deployment {
	meta, l := getDiscoveryMeta(tc)
	return &appsv1.Deployment{
		ObjectMeta: meta,
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: l.LabelSelector(),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: l.Labels(),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: meta.Name,
					Containers: []corev1.Container{{
						Name:            "discovery",
						Image:           "pingcap/tidb-operator:v1.0.5",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command: []string{
							"/usr/local/bin/tidb-discovery",
						},
						Env: []corev1.EnvVar{
							{
								Name: "MY_POD_NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							},
						},
					}},
				},
			},
		},
	}
}

func getDiscoveryMeta(tc *v1alpha1.TidbCluster) (metav1.ObjectMeta, label.Label) {
	instanceName := tc.GetLabels()[label.InstanceLabelKey]
	discoveryLabel := label.New().Instance(instanceName)
	discoveryLabel["app.kubernetes.io/component"] = "discovery"

	objMeta := metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-discovery", tc.Name),
		Namespace: tc.Namespace,
		Labels:    discoveryLabel,
	}
	return objMeta, discoveryLabel
}
