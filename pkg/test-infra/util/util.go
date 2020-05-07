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
	"strings"

	"github.com/ngaut/log"
	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

// PDAddress ...
func PDAddress(tc *v1alpha1.TidbCluster) string {
	pdSvcName := controller.PDMemberName(tc.Name)
	return fmt.Sprintf("http://%s.%s.svc:2379", pdSvcName, tc.Namespace)
}

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
	if component == string(clusterTypes.PD) {
		priorityPort = 2379
	} else if component == string(clusterTypes.TiKV) {
		priorityPort = 20160
	} else if component == string(clusterTypes.TiDB) {
		priorityPort = 4000
	} else if component == string(clusterTypes.MySQL) {
		priorityPort = 3306
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

// BuildBinlogImage returns the binlog image name
func BuildBinlogImage(name string) string {
	var (
		b       strings.Builder
		version = fixture.Context.TiDBClusterConfig.ImageVersion
	)

	if fixture.Context.BinlogConfig.BinlogVersion != "" {
		version = fixture.Context.BinlogConfig.BinlogVersion
	}
	if fixture.Context.HubAddress != "" {
		fmt.Fprintf(&b, "%s/", fixture.Context.HubAddress)
	}

	b.WriteString(fixture.Context.DockerRepository)
	b.WriteString("/")
	b.WriteString(name)
	b.WriteString(":")
	b.WriteString(version)
	return b.String()
}
