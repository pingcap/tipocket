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

package util

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/ngaut/log"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

// FindPort get possible correct port when there are multiple ports
func FindPort(podName, component string, containers []corev1.Container) int32 {
	var container *corev1.Container
	for idx, c := range containers {
		if c.Name == component {
			container = &containers[idx]
			break
		}
	}
	// if we cannot find the target container according to component name, fallback to the first container
	if container == nil {
		log.Errorf("failed to find the main container %s in %s", component, podName)
		container = &containers[0]
	}
	ports := container.Ports
	var priorityPort int32 = 0
	if component == string(cluster.PD) {
		priorityPort = 2379
	} else if component == string(cluster.TiKV) {
		priorityPort = 20160
	} else if component == string(cluster.TiDB) {
		priorityPort = 4000
	} else if component == string(cluster.DM) {
		priorityPort = 8261
	} else if component == string(cluster.MySQL) {
		priorityPort = 3306
	} else if component == string(cluster.TiCDC) {
		priorityPort = 8301
	}

	for _, port := range ports {
		if port.ContainerPort == priorityPort {
			return priorityPort
		}
	}

	return ports[0].ContainerPort
}

// RenderTemplateFunc ...
func RenderTemplateFunc(tpl *template.Template, model interface{}) (string, error) {
	buff := new(bytes.Buffer)
	err := tpl.Execute(buff, model)
	if err != nil {
		return "", err
	}
	return buff.String(), nil
}

// ApplyObject applies k8s object
func ApplyObject(client client.Client, object runtime.Object) error {
	if err := client.Create(context.TODO(), object); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}
	return nil
}

// BuildImage builds a image URL: ${fixture.Context.HubAddress}/${fixture.Context.DockerRepository}/$name:$tag
// or returns the fullImageIfNotEmpty if it's not empty
func BuildImage(name, tag, fullImageIfNotEmpty string) string {
	if len(fullImageIfNotEmpty) > 0 {
		return fullImageIfNotEmpty
	}
	var b strings.Builder
	hub := chooseHub(name)
	if hub != "" {
		fmt.Fprintf(&b, "%s/", hub)
	}
	b.WriteString(fixture.Context.DockerRepository)
	b.WriteString("/")
	b.WriteString(name)
	b.WriteString(":")
	b.WriteString(tag)

	return b.String()
}

func chooseHub(image string) string {
	switch image {
	case "tidb":
		if fixture.Context.TiDBClusterConfig.TiDBHubAddress != "" {
			return fixture.Context.TiDBClusterConfig.TiDBHubAddress
		}
	case "tikv":
		if fixture.Context.TiDBClusterConfig.TiKVHubAddress != "" {
			return fixture.Context.TiDBClusterConfig.TiKVHubAddress
		}
	case "pd":
		if fixture.Context.TiDBClusterConfig.PDHubAddress != "" {
			return fixture.Context.TiDBClusterConfig.PDHubAddress
		}
	case "tiflash":
		if fixture.Context.TiDBClusterConfig.TiFlashHubAddress != "" {
			return fixture.Context.TiDBClusterConfig.TiFlashHubAddress
		}
	}
	return fixture.Context.HubAddress
}

// GetNodeIPsFromPod gets the IPs (or addresses) for nodes.
func GetNodeIPsFromPod(cli client.Client, namespace string, podLabels map[string]string) ([]string, error) {
	var ips []string
	pods := &corev1.PodList{}
	if err := cli.List(context.Background(), pods, client.InNamespace(namespace), client.MatchingLabels(podLabels)); err != nil {
		return ips, err
	}

	for _, item := range pods.Items {
		ips = append(ips, item.Status.HostIP)
	}
	return ips, nil
}

// GetServiceByMeta gets the service by its meta.
func GetServiceByMeta(cli client.Client, svc *corev1.Service) (*corev1.Service, error) {
	clone := svc.DeepCopy()
	key, err := client.ObjectKeyFromObject(clone)
	if err != nil {
		return nil, err
	}

	if err := cli.Get(context.Background(), key, clone); err != nil {
		return nil, err
	}
	return clone, nil
}

// IsInK8sPodEnvironment checks whether in the k8s pod environment
// refer: https://stackoverflow.com/questions/36639062/how-do-i-tell-if-my-container-is-running-inside-a-kubernetes-cluster/54130803#54130803
func IsInK8sPodEnvironment() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

// GetFQDNFromStsPod generates the full qualified domain name from a pod of sts which has a headless(or other service kinds) service with it
func GetFQDNFromStsPod(pod *corev1.Pod) (string, error) {
	if pod.Spec.Hostname == "" {
		return "", fmt.Errorf("expect non-empty .spec.hostname of %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	}
	if pod.Spec.Subdomain == "" {
		return "", fmt.Errorf("expect non-empty .spec.subdomain of %s/%s", pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
	}
	return fmt.Sprintf("%s.%s.%s.svc", pod.Spec.Hostname, pod.Spec.Subdomain, pod.Namespace), nil
}
