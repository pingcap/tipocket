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
	"log"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"

	"github.com/pingcap/tipocket/pkg/test-infra/binlog"
	"github.com/pingcap/tipocket/pkg/test-infra/br"
	"github.com/pingcap/tipocket/pkg/test-infra/cdc"
	"github.com/pingcap/tipocket/pkg/test-infra/chaos"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/mysql"
	"github.com/pingcap/tipocket/pkg/test-infra/tidb"

	corev1 "k8s.io/api/core/v1"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// E2eCli contains clients
type E2eCli struct {
	Config *rest.Config
	Cli    client.Client
	Chaos  *chaos.Chaos
	BR     *br.BrOps
	CDC    *cdc.CdcOps
	MySQL  *mysql.MySQLOps
	TiDB   *tidb.TidbOps
	Binlog *binlog.Ops
}

// NewE2eCli creates e2e client
func NewE2eCli(conf *rest.Config) *E2eCli {
	kubeCli, err := fixture.BuildGenericKubeClient(conf)
	if err != nil {
		e2elog.Failf("error creating kube-client: %v", err)
	}
	tidbClient := tidb.New(kubeCli)
	return &E2eCli{
		Config: conf,
		Cli:    kubeCli,
		Chaos:  chaos.New(kubeCli),
		BR:     br.New(kubeCli),
		CDC:    cdc.New(kubeCli),
		MySQL:  mysql.New(kubeCli),
		TiDB:   tidbClient,
		Binlog: binlog.New(kubeCli, tidbClient),
	}
}

// GetNodes gets physical nodes
func (e *E2eCli) GetNodes() (*corev1.NodeList, error) {
	nodes := &corev1.NodeList{}
	err := e.Cli.List(context.TODO(), nodes)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// CreateNamespace creates the specified namespace if not exist.
func (e *E2eCli) CreateNamespace(name string) error {
	ns := &corev1.Namespace{}
	if err := e.Cli.Get(context.TODO(), types.NamespacedName{Name: name}, ns); err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Namespace %s doesn't exist. Creating...", name)
			if err = e.Cli.Create(context.TODO(), &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			}); err != nil {
				return err
			}
		}
		return err
	}
	return nil
}
