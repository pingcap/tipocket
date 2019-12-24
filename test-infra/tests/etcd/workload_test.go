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

package etcd

import (
	"github.com/pingcap/tipocket/test-infra/pkg/chaos"
	"time"

	"github.com/onsi/ginkgo"

	"github.com/pingcap/tipocket/test-infra/pkg/etcd"
	"github.com/pingcap/tipocket/test-infra/pkg/fixture"

	_ "github.com/pingcap/tidb/types/parser_driver"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
)

var _ = ginkgo.Describe("etcd", func() {
	// FIXME: framework will check if all nodes are ready for 3 minutes after each spec, which
	// is time consuming
	f := framework.NewDefaultFramework("etcd")
	f.SkipPrivilegedPSPBinding = true

	var ns string
	var c *Client
	var kubeCli *kubernetes.Clientset

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		e2elog.Logf("Working namespace %s", ns)
		var err error
		kubeCli, err = framework.LoadClientset()
		framework.ExpectNoError(err, "Expected to load kubernetes clientset.")
		conf, err := framework.LoadConfig()
		framework.ExpectNoError(err, "Expected to load config.")
		c = newClient(conf)
	})

	ginkgo.It("should tolerant random pod kill chaos", func() {
		name := "etcd-pod-kill"

		// ETCD source
		et, err := c.etcd.ApplyETCD(&etcd.ETCDSpec{
			Name:      name,
			Namespace: ns,
			Version:   etcdVersion,
			Replicas:  etcdReplicas,
			Storage:   fixture.StorageTypeLocal,
		})

		framework.ExpectNoError(err, "Expected to deploy etcd.")

		err = framework.WaitForStatefulSetReplicasReady(et.Sts.Name, et.Sts.Namespace, kubeCli, 10*time.Second, 5*time.Minute)
		framework.ExpectNoError(err, "Expected etcd ready.")

		nemesis := nemesisMap[fixture.E2eContext.Nemesis]
		err = nemesis(c.chaos, ns)
		framework.ExpectNoError(err, "Expected to apply nemesis.")

	})
})

type Client struct {
	chaos *chaos.Chaos
	etcd  *etcd.ETCDOps
}

func newClient(conf *rest.Config) *Client {
	kubeCli, err := fixture.BuildGenericKubeClient(conf)
	if err != nil {
		e2elog.Failf("error creating kube-client: %v", err)
	}
	return &Client{
		etcd:  etcd.New(kubeCli),
		chaos: chaos.New(kubeCli),
	}
}
