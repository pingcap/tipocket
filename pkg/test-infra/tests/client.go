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
	"log"
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // auth in cluster
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/pingcap/tipocket/pkg/test-infra/abtest"
	"github.com/pingcap/tipocket/pkg/test-infra/binlog"
	"github.com/pingcap/tipocket/pkg/test-infra/cdc"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/mysql"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestClient encapsulates many kinds of clients
var TestClient *TestCli

// TestCli contains clients
type TestCli struct {
	Config *rest.Config
	Cli    client.Client
	CDC    *cdc.CdcOps
	MySQL  *mysql.MySQLOps
	TiDB   *tidb.TidbOps
	Binlog *binlog.Ops
	ABTest *abtest.Ops
}

func newTestCli(conf *rest.Config) *TestCli {
	kubeCli, err := fixture.BuildGenericKubeClient(conf)
	if err != nil {
		log.Fatalf("error creating kube-client: %v", err)
	}
	tidbClient := tidb.New(kubeCli)
	return &TestCli{
		Config: conf,
		Cli:    kubeCli,
		CDC:    cdc.New(kubeCli),
		MySQL:  mysql.New(kubeCli),
		TiDB:   tidbClient,
		Binlog: binlog.New(kubeCli, tidbClient),
		ABTest: abtest.New(kubeCli, tidbClient),
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

// CreateNamespace creates the specified namespace if not exist.
func (e *TestCli) CreateNamespace(name string) error {
	ns := &corev1.Namespace{}
	if err := e.Cli.Get(context.TODO(), types.NamespacedName{Name: name}, ns); err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Namespace %s doesn't exist. Creating...", name)
			if err = e.Cli.Create(context.TODO(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
					// admission-webhook is needed for injecting io chaos
					Labels: map[string]string{
						"app.kubernetes.io/component": "webhook",
					},
				},
			}); err != nil {
				return err
			}
		}
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
		log.Fatalf("build config failed: %+v", err)
	}
	TestClient = newTestCli(conf)
}
