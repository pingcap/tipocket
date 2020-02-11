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
	"context"

	"k8s.io/client-go/rest"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/br"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/cdc"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/chaos"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/mysql"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/tidb"

	corev1 "k8s.io/api/core/v1"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type E2eCli struct {
	Config *rest.Config
	Cli    client.Client
	Chaos  *chaos.Chaos
	BR     *br.BrOps
	CDC    *cdc.CdcOps
	MySQL  *mysql.MySQLOps
	TiDB   *tidb.TidbOps
}

func NewE2eCli(conf *rest.Config) *E2eCli {
	kubeCli, err := fixture.BuildGenericKubeClient(conf)
	if err != nil {
		e2elog.Failf("error creating kube-client: %v", err)
	}
	return &E2eCli{
		Config: conf,
		Cli:    kubeCli,
		Chaos:  chaos.New(kubeCli),
		BR:     br.New(kubeCli),
		CDC:    cdc.New(kubeCli),
		MySQL:  mysql.New(kubeCli),
		TiDB:   tidb.New(kubeCli),
	}
}

func (e *E2eCli) GetNodes() (*corev1.NodeList, error) {
	nodes := &corev1.NodeList{}
	err := e.Cli.List(context.TODO(), nodes)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}
