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
	"github.com/pingcap/tipocket/test-infra/pkg/br"
	"github.com/pingcap/tipocket/test-infra/pkg/cdc"
	"github.com/pingcap/tipocket/test-infra/pkg/chaos"
	"github.com/pingcap/tipocket/test-infra/pkg/fixture"
	"github.com/pingcap/tipocket/test-infra/pkg/mysql"
	"github.com/pingcap/tipocket/test-infra/pkg/tidb"
	"k8s.io/client-go/rest"

	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
)

type E2eCli struct {
	Chaos *chaos.Chaos
	BR    *br.BrOps
	CDC   *cdc.CdcOps
	MySQL *mysql.MySQLOps
	TiDB  *tidb.TidbOps
}

func NewE2eCli(conf *rest.Config) *E2eCli {
	kubeCli, err := fixture.BuildGenericKubeClient(conf)
	if err != nil {
		e2elog.Failf("error creating kube-client: %v", err)
	}
	return &E2eCli{
		Chaos: chaos.New(kubeCli),
		BR:    br.New(kubeCli),
		CDC:   cdc.New(kubeCli),
		MySQL: mysql.New(kubeCli),
		TiDB:  tidb.New(kubeCli),
	}
}
