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

package cdc

import (
	"time"

	"github.com/onsi/ginkgo"
	chaosv1alpha1 "github.com/pingcap/chaos-operator/api/v1alpha1"
	"github.com/pingcap/tipocket/pkg/cdc"
	"github.com/pingcap/tipocket/pkg/fixture"
	"github.com/pingcap/tipocket/pkg/mysql"
	"github.com/pingcap/tipocket/pkg/tidb"
	"github.com/pingcap/tipocket/tests/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
)

var _ = ginkgo.Describe("cdc", func() {

	// FIXME: framework will check if all nodes are ready for 3 minutes after each spec, which
	// is time consuming
	f := framework.NewDefaultFramework("cdc")
	f.SkipPrivilegedPSPBinding = true

	var ns string
	var c *util.E2eCli

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		conf, err := framework.LoadConfig()
		framework.ExpectNoError(err, "Expected to load config.")
		c = util.NewE2eCli(conf)
	})

	ginkgo.It("should tolerant random pod kill chaos", func() {
		name := "cdc-pod-kill"

		// CDC source
		tc := tidb.RecommendedTiDBCluster(ns, name).Make()
		err := c.TiDB.ApplyTiDBCluster(tc)
		framework.ExpectNoError(err, "Expected to deploy tidb-cluster.")

		e2elog.Logf("Wait TiDB cluster ready")
		err = c.TiDB.WaitTiDBClusterReady(tc, 5*time.Minute)
		framework.ExpectNoError(err, "Expected to see tidb-cluster is ready.")

		e2elog.Logf("Deploy CDC")
		// CDC sink
		mysql, err := c.MySQL.ApplyMySQL(&mysql.MySQLSpec{
			Namespace: ns,
			Name:      name,
			Resource:  fixture.Medium,
			Storage:   fixture.StorageTypeLocal,
		})
		framework.ExpectNoError(err, "Expected to deploy mysql.")

		cc, err := c.CDC.ApplyCDC(&cdc.CDCSpec{
			Namespace: ns,
			Name:      name,
			Resources: fixture.Small,
			Replicas:  3,
			Source:    tc,
		})
		framework.ExpectNoError(err, "Expected to deploy CDC.")

		_ = &cdc.CDCJob{
			CDC:     cc,
			SinkURI: mysql.URI(),
		}
		//err = c.CDC.StartJob(&cdc.CDCJob{
		//	CDC:     cc,
		//	SinkURI: mysql.URI(),
		//})
		//framework.ExpectNoError(err, "Expected to start a CDC job.")

		// TODO: able to describe describe chaos in BDD-style
		podKill := &chaosv1alpha1.PodChaos{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: chaosv1alpha1.PodChaosSpec{
				Selector: chaosv1alpha1.SelectorSpec{
					// Randomly kill in namespace
					Namespaces: []string{ns},
				},
				Mode:     chaosv1alpha1.OnePodMode,
				Duration: "3m",
			},
		}
		err = c.Chaos.ApplyPodChaos(podKill)
		framework.ExpectNoError(err, "Expected to apply pod chaos.")

		// TODO: write data to TiDB

		// check chaos tolerance for 5 minutes
		err = wait.PollImmediate(10*time.Second, 5*time.Minute, func() (bool, error) {
			// TODO: check acceptance of CDC
			klog.Info("Dummy waiting...")
			return false, nil
		})
		if err != wait.ErrWaitTimeout {
			e2elog.Failf("Encounter unexpected error: %v", err)
		}
	})
})
