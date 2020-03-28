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
	"io/ioutil"
	"time"

	"github.com/ngaut/log"
	"golang.org/x/sync/errgroup"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pingcap/errors"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/label"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
)

const (
	tikvDir     = "/var/lib/tikv"
	tikvDataDir = "/var/lib/tikv/data"
	pdDir       = "/var/lib/pd"
	pdDataDir   = "/var/lib/pd/data"

	ioChaosAnnotation = "admission-webhook.pingcap.com/request"
)

// TidbOps knows how to operate TiDB on k8s
type TidbOps struct {
	cli client.Client
}

// Config of tidb/tikv/pd on applying TiDB cluster
type Config struct {
	Tidb string
	Tikv string
	Pd   string
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

func (t *TidbOps) GetNodes(tc *Recommendation) ([]clusterTypes.Node, error) {
	pods := &corev1.PodList{}
	if err := t.cli.List(context.TODO(), pods, &client.ListOptions{Namespace: tc.NS},
		client.MatchingLabels{"app.kubernetes.io/instance": tc.Name}); err != nil {
		return []clusterTypes.Node{}, err
	}

	r := pods.DeepCopy()
	r.Items = []corev1.Pod{}
	for _, pod := range pods.Items {
		r.Items = append(r.Items, pod)
	}
	return t.parseNodeFromPodList(r), nil
}

func (t *TidbOps) GetClientNodes(tc *Recommendation) ([]clusterTypes.ClientNode, error) {
	var clientNodes []clusterTypes.ClientNode

	k8sNodes, err := t.GetK8sNodes()
	if err != nil {
		return clientNodes, err
	}
	ip := getNodeIP(k8sNodes)
	if ip == "" {
		return clientNodes, errors.New("k8s node not found")
	}

	svc, err := t.GetTiDBServiceByMeta(&tc.Service.ObjectMeta)
	if err != nil {
		return clientNodes, err
	}
	clientNodes = append(clientNodes, clusterTypes.ClientNode{
		Namespace:   svc.ObjectMeta.Namespace,
		ClusterName: svc.ObjectMeta.Labels["app.kubernetes.io/instance"],
		IP:          ip,
		Port:        getTiDBNodePort(svc),
	})
	return clientNodes, nil
}

// GetK8sNodes gets physical nodes
func (t *TidbOps) GetK8sNodes() (*corev1.NodeList, error) {
	nodes := &corev1.NodeList{}
	err := t.cli.List(context.TODO(), nodes)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (t *TidbOps) GetTiDBCluster(ns, name string) (*v1alpha1.TidbCluster, error) {
	tc := &v1alpha1.TidbCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	key, err := client.ObjectKeyFromObject(tc)
	if err != nil {
		return nil, err
	}

	if err := t.cli.Get(context.TODO(), key, tc); err != nil {
		return nil, err
	}

	return tc, nil
}

func (t *TidbOps) ApplyTiDBCluster(cluster *Recommendation, config ...Config) error {
	if len(config) > 1 {
		return errors.Errorf("too many config args: expected 1, but %d got", len(config))
	}

	tc := cluster.TidbCluster
	tm := cluster.TidbMonitor
	c := &Config{
		Tikv: fixture.Context.TiKVConfigFile,
		Pd:   fixture.Context.PDConfigFile,
		Tidb: fixture.Context.TiDBConfigFile,
	}
	if len(config) == 1 {
		c = &config[0]
	}
	desired := tc.DeepCopy()

	log.Info("Apply tidb discovery")
	if err := t.applyDiscovery(tc); err != nil {
		return err
	}
	log.Info("Apply tidb configmap")
	if err := t.applyTiDBConfigMap(tc, c.Tidb); err != nil {
		return err
	}
	log.Info("Apply pd configmap")
	if err := t.applyPDConfigMap(tc, c.Pd); err != nil {
		return err
	}
	log.Info("Apply tikv configmap")
	if err := t.applyTiKVConfigMap(tc, c.Tikv); err != nil {
		return err
	}
	if tc.Spec.Pump != nil {
		log.Info("Apply pump configmap")
		if err := t.applyPumpConfigMap(tc); err != nil {
			return err
		}
	}
	// apply tc
	_, err := controllerutil.CreateOrUpdate(context.TODO(), t.cli, tc, func() error {
		tc.Spec = desired.Spec
		tc.Annotations = desired.Annotations
		tc.Labels = desired.Labels
		return nil
	})
	if err = t.waitTiDBClusterReady(tc, fixture.Context.WaitClusterReadyDuration); err != nil {
		return err
	}
	return t.applyTiDBMonitor(tm)
}

func (t *TidbOps) waitTiDBClusterReady(tc *v1alpha1.TidbCluster, timeout time.Duration) error {
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
			log.Errorf("error getting tidbcluster: %v", err)
			return false, nil
		}
		if local.Status.PD.StatefulSet == nil {
			return false, nil
		}
		pdReady, pdDesired := local.Status.PD.StatefulSet.ReadyReplicas, local.Spec.PD.Replicas
		if pdReady < pdDesired {
			log.Infof("PD do not have enough ready replicas, ready: %d, desired: %d", pdReady, pdDesired)
			return false, nil
		}
		if local.Status.TiKV.StatefulSet == nil {
			return false, nil
		}
		tikvReady, tikvDesired := local.Status.TiKV.StatefulSet.ReadyReplicas, local.Spec.TiKV.Replicas
		if tikvReady < tikvDesired {
			log.Infof("TiKV do not have enough ready replicas, ready: %d, desired: %d", tikvReady, tikvDesired)
			return false, nil
		}
		if local.Status.TiDB.StatefulSet == nil {
			return false, nil
		}
		tidbReady, tidbDesired := local.Status.TiDB.StatefulSet.ReadyReplicas, local.Spec.TiDB.Replicas
		if tidbReady < tidbDesired {
			log.Infof("TiDB do not have enough ready replicas, ready: %d, desired: %d", tidbReady, tidbDesired)
			return false, nil
		}
		return true, nil
	})
}

func (t *TidbOps) applyTiDBMonitor(tm *v1alpha1.TidbMonitor) error {
	desired := tm.DeepCopy()
	_, err := controllerutil.CreateOrUpdate(context.TODO(), t.cli, tm, func() error {
		tm.Spec = desired.Spec
		tm.Annotations = desired.Annotations
		tm.Labels = desired.Labels
		return nil
	})
	return err
}

func (t *TidbOps) Delete(tc *Recommendation) error {
	var g errgroup.Group
	g.Go(func() error {
		err := t.cli.Delete(context.TODO(), tc.TidbCluster)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		err := t.cli.Delete(context.TODO(), tc.TidbMonitor)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		return nil
	})
	return g.Wait()
}

func (t *TidbOps) applyTiDBConfigMap(tc *v1alpha1.TidbCluster, configFile string) error {
	configMap, err := getTiDBConfigMap(tc)
	if err != nil {
		return err
	}
	if configMap.Data["config-file"], err = readFileAsString(configFile); err != nil {
		return err
	}
	desired := configMap.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), t.cli, configMap, func() error {
		configMap.Data = desired.Data
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (t *TidbOps) applyPDConfigMap(tc *v1alpha1.TidbCluster, configFile string) error {
	configMap, err := getPDConfigMap(tc)
	if err != nil {
		return err
	}
	if configMap.Data["config-file"], err = readFileAsString(configFile); err != nil {
		return err
	}
	desired := configMap.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), t.cli, configMap, func() error {
		configMap.Data = desired.Data
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (t *TidbOps) applyTiKVConfigMap(tc *v1alpha1.TidbCluster, configFile string) error {
	configMap, err := getTiKVConfigMap(tc)
	if err != nil {
		return err
	}
	if configMap.Data["config-file"], err = readFileAsString(configFile); err != nil {
		return err
	}
	desired := configMap.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), t.cli, configMap, func() error {
		configMap.Data = desired.Data
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (t *TidbOps) applyPumpConfigMap(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Pump == nil {
		return nil
	}
	configMap, err := getPumpConfigMap(tc)
	if err != nil {
		return err
	}
	desired := configMap.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), t.cli, configMap, func() error {
		configMap.Data = desired.Data
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (t *TidbOps) applyTiDBService(s *corev1.Service) error {
	err := t.cli.Create(context.TODO(), s)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (t *TidbOps) applyDiscovery(tc *v1alpha1.TidbCluster) error {
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
			log.Warningf("error getting tidbcluster: %v", err)
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

func readFileAsString(filename string) (string, error) {
	if filename == "" {
		return ``, nil
	}
	if bytes, err := ioutil.ReadFile(filename); err != nil {
		return ``, err
	} else {
		return string(bytes), nil
	}
}

func getPDConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	scriptModel := &PDStartScriptModel{DataDir: pdDir}
	if getIOChaosAnnotation(tc, "pd") == "chaosfs-pd" {
		scriptModel.DataDir = pdDataDir
	}
	s, err := RenderPDStartScript(scriptModel)
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
		},
	}, nil
}

func getTiKVConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	scriptModel := &TiKVStartScriptModel{DataDir: tikvDir}
	if getIOChaosAnnotation(tc, "tikv") == "chaosfs-tikv" {
		scriptModel.DataDir = tikvDataDir
	}
	s, err := RenderTiKVStartScript(scriptModel)
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
		},
	}, nil
}

func getPumpConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	c, err := RenderPumpConfig(&PumpConfigModel{})
	if err != nil {
		return nil, err
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: tc.Namespace,
			Name:      fmt.Sprintf("%s-pump", tc.Name),
		},
		Data: map[string]string{
			"pump-config": c,
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
						Image:           "pingcap/tidb-operator:v1.0.6",
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

func (t *TidbOps) parseNodeFromPodList(pods *corev1.PodList) []clusterTypes.Node {
	var nodes []clusterTypes.Node
	for _, pod := range pods.Items {
		component, ok := pod.ObjectMeta.Labels["app.kubernetes.io/component"]
		if !ok {
			component = ""
		} else if component == "discovery" || component == "monitor" {
			continue
		}

		nodes = append(nodes, clusterTypes.Node{
			Namespace: pod.ObjectMeta.Namespace,
			// TODO use better way to retrieve version?
			Version:   fixture.Context.ImageVersion,
			PodName:   pod.ObjectMeta.Name,
			IP:        pod.Status.PodIP,
			Component: clusterTypes.Component(component),
			Port:      util.FindPort(pod.ObjectMeta.Name, pod.Spec.Containers[0].Ports),
			Client: &clusterTypes.Client{
				Namespace:   pod.ObjectMeta.Namespace,
				ClusterName: pod.ObjectMeta.Labels["app.kubernetes.io/instance"],
				PDMemberFunc: func(ns, name string) (string, []string, error) {
					return t.GetPDMember(ns, name)
				},
			},
		})
	}
	return nodes
}

func getNodeIP(nodeList *corev1.NodeList) string {
	if len(nodeList.Items) == 0 {
		return ""
	}
	return nodeList.Items[0].Status.Addresses[0].Address
}

func getTiDBNodePort(svc *corev1.Service) int32 {
	for _, port := range svc.Spec.Ports {
		if port.Port == 4000 {
			return port.NodePort
		}
	}
	return 0
}

func getIOChaosAnnotation(tc *v1alpha1.TidbCluster, component string) string {
	switch component {
	case "tikv":
		if tc.Spec.TiKV.Annotations != nil {
			if s, ok := tc.Spec.TiKV.Annotations[ioChaosAnnotation]; ok {
				return s
			}
		}
	case "pd":
		if tc.Spec.PD.Annotations != nil {
			if s, ok := tc.Spec.PD.Annotations[ioChaosAnnotation]; ok {
				return s
			}
		}
	}

	return ""
}
