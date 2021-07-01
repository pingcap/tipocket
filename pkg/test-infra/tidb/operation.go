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
	b64 "encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"
	"golang.org/x/sync/errgroup"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/tests"
	"github.com/pingcap/tipocket/pkg/test-infra/util"
	"github.com/pingcap/tipocket/pkg/tidb-operator/apis/pingcap/v1alpha1"
	"github.com/pingcap/tipocket/pkg/tidb-operator/label"
)

const (
	tikvDir     = "/var/lib/tikv"
	tikvDataDir = "/var/lib/tikv/data"
	pdDir       = "/var/lib/pd"
	pdDataDir   = "/var/lib/pd/data"
	// used for tikv data encryption
	tikvEncryptionMasterKey = "c7fd825f4ec91c07067553896cb1b4ad9e32e9175e7750aa39cc1771fc8eb589"
	plaintextProtocolHeader = "plaintext://"
)

// Ops knows how to operate TiDB
type Ops struct {
	cli    client.Client
	tc     *Recommendation
	config fixture.TiDBClusterConfig
	ns     string
	name   string
}

// New ...
func New(namespace, name string, config fixture.TiDBClusterConfig) *Ops {
	return &Ops{cli: tests.TestClient.Cli, tc: RecommendedTiDBCluster(namespace, name, config),
		ns: namespace, name: name, config: config}
}

// GetTiDBCluster ...
func (o *Ops) GetTiDBCluster() *v1alpha1.TidbCluster {
	return o.tc.TidbCluster
}

func (o *Ops) getTiDBServiceByClusterName(ns string, clusterName string) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      fmt.Sprintf("%s-tidb", clusterName),
		},
	}
	key, err := client.ObjectKeyFromObject(svc)
	if err != nil {
		return nil, err
	}

	if err := o.cli.Get(context.TODO(), key, svc); err != nil {
		return nil, err
	}

	return svc, nil
}

// GetNodes ...
func (o *Ops) GetNodes() ([]cluster.Node, error) {
	pods := &corev1.PodList{}
	if err := o.cli.List(context.TODO(), pods, &client.ListOptions{Namespace: o.ns},
		client.MatchingLabels{"app.kubernetes.io/instance": o.name}); err != nil {
		return []cluster.Node{}, err
	}

	r := pods.DeepCopy()
	r.Items = []corev1.Pod{}
	for _, pod := range pods.Items {
		r.Items = append(r.Items, pod)
	}
	return o.parseNodeFromPodList(r)
}

// GetClientNodes ...
func (o *Ops) GetClientNodes() ([]cluster.ClientNode, error) {
	var clientNodes []cluster.ClientNode

	if util.IsInK8sPodEnvironment() {
		svc, err := o.getTiDBServiceByClusterName(o.ns, o.name)
		if err != nil {
			return nil, err
		}
		clientNodes = append(clientNodes, cluster.ClientNode{
			Namespace:   o.ns,
			Component:   "tidb",
			ClusterName: o.name,
			IP:          svc.Spec.ClusterIP,
			Port:        getTiDBServicePort(svc),
		})
	} else {
		// If case isn't running on k8s pod, uses nodeIP:tidb_port as clientNode for conveniently debug on local
		ips, err := util.GetNodeIPsFromPod(o.cli, o.ns, map[string]string{"app.kubernetes.io/instance": o.name})
		if err != nil {
			return nil, err
		} else if len(ips) == 0 {
			return nil, errors.New("k8s node not found")
		}
		svc, err := o.getTiDBServiceByClusterName(o.ns, o.name)
		if err != nil {
			return clientNodes, err
		}
		clientNodes = append(clientNodes, cluster.ClientNode{
			Namespace:   o.ns,
			ClusterName: o.name,
			Component:   "tidb",
			IP:          ips[0],
			Port:        getTiDBNodePort(svc),
		})
	}
	return clientNodes, nil
}

// getK8sNodes gets physical nodes
func (o *Ops) getK8sNodes() (*corev1.NodeList, error) {
	nodes := &corev1.NodeList{}
	err := o.cli.List(context.TODO(), nodes)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// Apply ...
func (o *Ops) Apply() error {
	tc := o.tc.TidbCluster
	tm := o.tc.TidbMonitor
	desired := tc.DeepCopy()

	log.Info("Apply tidb configmap")
	if err := o.applyTiDBConfigMap(tc, o.config.TiDBConfig); err != nil {
		return err
	}
	log.Info("Apply pd configmap")
	if err := o.applyPDConfigMap(tc, o.config.PDConfig); err != nil {
		return err
	}
	log.Info("Apply tikv configmap")
	if err := o.applyTiKVConfigMap(tc, o.config.TiKVConfig); err != nil {
		return err
	}
	if tc.Spec.Pump != nil {
		log.Info("Apply pump configmap")
		if err := o.applyPumpConfigMap(tc); err != nil {
			return err
		}
	}
	// apply tc
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), o.cli, tc, func() error {
		tc.Spec = desired.Spec
		tc.Annotations = desired.Annotations
		tc.Labels = desired.Labels
		return nil
	}); err != nil {
		return err
	}
	if err := o.waitTiDBReady(tc, fixture.Context.WaitClusterReadyDuration); err != nil {
		return err
	}
	return o.applyTiDBMonitor(tm)
}

func (o *Ops) waitTiDBReady(tc *v1alpha1.TidbCluster, timeout time.Duration) error {
	name := tc.Name
	namespace := tc.Namespace
	local := tc.DeepCopy()
	return wait.PollImmediate(5*time.Second, timeout, func() (bool, error) {
		key, err := client.ObjectKeyFromObject(local)
		if err != nil {
			return false, err
		}
		if err = o.cli.Get(context.TODO(), key, local); err != nil {
			if errors.IsNotFound(err) {
				return false, err
			}
			log.Errorf("error getting TidbOps: %v", err)
			return false, nil
		}

		if local.Status.PD.StatefulSet == nil {
			return false, nil
		}
		pdReady, pdDesired := local.Status.PD.StatefulSet.ReadyReplicas, local.Spec.PD.Replicas
		if pdReady < pdDesired {
			log.Infof("PD[%s/%s] do not have enough ready replicas, ready: %d, desired: %d", namespace, name, pdReady, pdDesired)
			return false, nil
		}
		if local.Status.TiKV.StatefulSet == nil {
			return false, nil
		}
		tikvReady, tikvDesired := local.Status.TiKV.StatefulSet.ReadyReplicas, local.Spec.TiKV.Replicas
		if tikvReady < tikvDesired {
			log.Infof("TiKV[%s/%s] do not have enough ready replicas, ready: %d, desired: %d", namespace, name, tikvReady, tikvDesired)
			return false, nil
		}
		if local.Status.TiDB.StatefulSet == nil {
			return false, nil
		}
		tidbReady, tidbDesired := local.Status.TiDB.StatefulSet.ReadyReplicas, local.Spec.TiDB.Replicas
		if tidbReady < tidbDesired {
			log.Infof("TiDB[%s/%s] do not have enough ready replicas, ready: %d, desired: %d", namespace, name, tidbReady, tidbDesired)
			return false, nil
		}
		if tc.Spec.TiFlash != nil {
			if local.Status.TiFlash.StatefulSet == nil {
				return false, nil
			}
			tiFlashReady, tiFlashDesired := local.Status.TiFlash.StatefulSet.ReadyReplicas, local.Spec.TiFlash.Replicas
			if tiFlashReady < tiFlashDesired {
				log.Infof("TiFlash do not have enough ready replicas, ready: %d, desired: %d", tiFlashReady, tiFlashDesired)
				return false, nil
			}
		}
		return true, nil
	})
}

func (o *Ops) applyTiDBMonitor(tm *v1alpha1.TidbMonitor) error {
	if tm == nil {
		return nil
	}
	desired := tm.DeepCopy()
	_, err := controllerutil.CreateOrUpdate(context.TODO(), o.cli, tm, func() error {
		tm.Spec = desired.Spec
		tm.Annotations = desired.Annotations
		tm.Labels = desired.Labels
		return nil
	})
	return err
}

// Delete ...
func (o *Ops) Delete() error {
	var g errgroup.Group
	g.Go(func() error {
		err := o.cli.Delete(context.TODO(), o.tc.TidbCluster)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		if o.tc.TidbMonitor == nil {
			return nil
		}
		err := o.cli.Delete(context.TODO(), o.tc.TidbMonitor)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		return nil
	})
	return g.Wait()
}

func (o *Ops) applyTiDBConfigMap(tc *v1alpha1.TidbCluster, configString string) error {
	configMap, err := getTiDBConfigMap(tc, o.config)
	if err != nil {
		return err
	}
	configData, err := parseConfig(configString)
	if err != nil {
		return err
	}
	configMap.Data["config-file"] = configData
	desired := configMap.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), o.cli, configMap, func() error {
		configMap.Data = desired.Data
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (o *Ops) applyPDConfigMap(tc *v1alpha1.TidbCluster, configString string) error {
	configMap, err := getPDConfigMap(tc)
	if err != nil {
		return err
	}
	configData, err := parseConfig(configString)
	if err != nil {
		return err
	}
	configMap.Data["config-file"] = configData
	desired := configMap.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), o.cli, configMap, func() error {
		configMap.Data = desired.Data
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (o *Ops) applyTiKVConfigMap(tc *v1alpha1.TidbCluster, configString string) error {
	configMap, err := getTiKVConfigMap(tc)
	if err != nil {
		return err
	}
	configData, err := parseConfig(configString)
	if err != nil {
		return err
	}
	configMap.Data["config-file"] = configData
	desired := configMap.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), o.cli, configMap, func() error {
		configMap.Data = desired.Data
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (o *Ops) applyPumpConfigMap(tc *v1alpha1.TidbCluster) error {
	if tc.Spec.Pump == nil {
		return nil
	}
	configMap, err := getPumpConfigMap(tc)
	if err != nil {
		return err
	}
	desired := configMap.DeepCopy()
	if _, err := controllerutil.CreateOrUpdate(context.TODO(), o.cli, configMap, func() error {
		configMap.Data = desired.Data
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (o *Ops) applyTiDBService(s *corev1.Service) error {
	err := o.cli.Create(context.TODO(), s)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// GetPDMember ...
func (o *Ops) GetPDMember(namespace, name string) (string, []string, error) {
	var local v1alpha1.TidbCluster
	var members []string
	err := wait.PollImmediate(5*time.Second, time.Minute*time.Duration(5), func() (bool, error) {
		key := types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		}
		err := o.cli.Get(context.TODO(), key, &local)
		if err != nil && errors.IsNotFound(err) {
			return false, err
		}
		if err != nil {
			log.Warningf("error getting TidbOps: %v", err)
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

func parseConfig(config string) (string, error) {
	if strings.HasPrefix(config, plaintextProtocolHeader) && len(config) > len(plaintextProtocolHeader) {
		return extractRawConfig(config), nil
	}
	// Parse config
	configData, err := readFileAsString(config)
	if err == nil {
		return configData, nil
	}
	// parse config string like "base64://BASE64CONTENT"
	// Supported scheme: base64
	u, err := url.Parse(config)
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "base64":
		// An workaround: When parse url like "base64://BASE64CONTENT", BASE64CONTENT would be parsed in url.Host
		configData, err = readBase64AsString(u.Host)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("not a supported scheme: %s", config)
		// Add more Scheme support here, like http
	}
	return configData, nil
}

func extractRawConfig(raw string) string {
	return raw[len(plaintextProtocolHeader):]
}

func readFileAsString(filename string) (string, error) {
	if filename == "" {
		return ``, nil
	}
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return ``, err
	}
	return string(bytes), nil
}

func readBase64AsString(base64config string) (string, error) {
	configData, err := b64.StdEncoding.DecodeString(base64config)
	if err != nil {
		return "", err
	}
	return string(configData), nil
}

func getPDConfigMap(tc *v1alpha1.TidbCluster) (*corev1.ConfigMap, error) {
	scriptModel := &PDStartScriptModel{DataDir: pdDir}
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

func getTiDBConfigMap(tc *v1alpha1.TidbCluster, config fixture.TiDBClusterConfig) (*corev1.ConfigMap, error) {
	s, err := RenderTiDBStartScript(&StartScriptModel{ClusterName: tc.Name, Failpoints: config.TiDBFailPoint})
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
	scriptModel := &TiKVStartScriptModel{DataDir: tikvDir, MasterKey: tikvEncryptionMasterKey}
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
	c, err := RenderPumpConfig(&pumpConfigModel{})
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

func (o *Ops) parseNodeFromPodList(pods *corev1.PodList) ([]cluster.Node, error) {
	var nodes []cluster.Node
	for _, pod := range pods.Items {
		component, ok := pod.ObjectMeta.Labels["app.kubernetes.io/component"]
		if !ok {
			component = ""
			continue
		} else if component == "discovery" || component == "monitor" {
			continue
		}
		var podIP = pod.Status.PodIP
		// because all tidb components are managed by statefulset with a related -peer headless service
		// we use the fqdn as the the node ip
		if component == "tikv" || component == "tidb" || component == "pd" || component == "tiflash" || component == "ticdc" {
			var err error
			podIP, err = util.GetFQDNFromStsPod(&pod)
			if err != nil {
				return nil, err
			}
		}
		nodes = append(nodes, cluster.Node{
			Namespace: pod.ObjectMeta.Namespace,
			PodName:   pod.ObjectMeta.Name,
			IP:        podIP,
			Component: cluster.Component(component),
			Port:      util.FindPort(pod.ObjectMeta.Name, component, pod.Spec.Containers),
			Client: &cluster.Client{
				Namespace:   pod.ObjectMeta.Namespace,
				ClusterName: pod.ObjectMeta.Labels["app.kubernetes.io/instance"],
				PDMemberFunc: func(ns, name string) (string, []string, error) {
					return o.GetPDMember(ns, name)
				},
			},
		})
	}
	return nodes, nil
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

func getTiDBServicePort(svc *corev1.Service) int32 {
	for _, port := range svc.Spec.Ports {
		if port.Port == 4000 {
			return port.Port
		}
	}
	panic("couldn't find the tidb exposed port")
}

// GetTiDBConfig is used for Matrix-related setups
func (o *Ops) GetTiDBConfig() *fixture.TiDBClusterConfig {
	return &o.config
}
