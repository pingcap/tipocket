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

	"github.com/pingcap/tidb-operator/pkg/apis/pingcap/v1alpha1"
	"github.com/pingcap/tidb-operator/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

// PDAddress ...
func PDAddress(tc *v1alpha1.TidbCluster) string {
	pdSvcName := controller.PDMemberName(tc.Name)
	return fmt.Sprintf("http://%s.%s.svc:2379", pdSvcName, tc.Namespace)
}

// FindPort get possible correct port when there are multiple ports
func FindPort(podName string, ports []corev1.ContainerPort) int32 {
	if len(ports) == 0 {
		return 0
	}

	var priorityPort int32 = 0
	if strings.Contains(podName, "pd") {
		priorityPort = 2379
	} else if strings.Contains(podName, "tikv") {
		priorityPort = 20160
	} else if strings.Contains(podName, "tidb") {
		priorityPort = 4000
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
