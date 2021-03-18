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

package tests

import (
	"context"
	"fmt"
	"os"

	"github.com/ngaut/log"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // auth in cluster
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestClient encapsulates many kinds of clients
var TestClient *TestCli

// TestCli contains clients
type TestCli struct {
	Cli client.Client
}

func newTestCli(conf *rest.Config) *TestCli {
	kubeCli, err := fixture.BuildGenericKubeClient(conf)
	if err != nil {
		log.Warnf("error creating kube-client: %v", err)
	}
	return &TestCli{
		Cli: kubeCli,
	}
}

// GetNodes gets physical nodes
func (e *TestCli) GetNodes() (*corev1.NodeList, error) {
	nodes := &corev1.NodeList{}
	err := e.Cli.List(context.TODO(), nodes)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// CreateNamespace creates the specified namespace
// and enable admission webhook if not exist.
func (e *TestCli) CreateNamespace(name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	if err := e.Cli.Get(context.TODO(), types.NamespacedName{Name: name}, ns); err == nil {
		return nil
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("get namespace %s failed: %+v", name, err)
	}

	if _, err := controllerutil.CreateOrUpdate(context.TODO(), e.Cli, ns, func() error {
		if ns.Labels != nil {
			ns.Labels["admission-webhook"] = "enabled"
		} else {
			ns.Labels = map[string]string{
				"admission-webhook": "enabled",
			}
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

// DeleteNamespace delete the specified namespace
func (e *TestCli) DeleteNamespace(name string) error {
	ns := &corev1.Namespace{}
	if err := e.Cli.Get(context.TODO(), types.NamespacedName{Name: name}, ns); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get namespace %s failed: %+v", name, err)
	}
	if err := e.Cli.Delete(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}); err != nil {
		return fmt.Errorf("delete namespace %s failed: %+v", name, err)
	}
	return nil
}

func init() {
	conf, err := clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	if err != nil {
		log.Warnf("build config failed: %+v", err)
	}
	TestClient = newTestCli(conf)
}
