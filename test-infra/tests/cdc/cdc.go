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
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	chaosv1alpha1 "github.com/pingcap/chaos-operator/api/v1alpha1"
	"github.com/pingcap/tipocket/pocket/core"
	"github.com/pingcap/tipocket/pocket/executor"
	"github.com/pingcap/tipocket/test-infra/pkg/cdc"
	"github.com/pingcap/tipocket/test-infra/pkg/fixture"
	"github.com/pingcap/tipocket/test-infra/pkg/mysql"
	"github.com/pingcap/tipocket/test-infra/pkg/tidb"
	putil "github.com/pingcap/tipocket/test-infra/pkg/util"
	"github.com/pingcap/tipocket/test-infra/tests/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"

	_ "github.com/pingcap/tidb/types/parser_driver"
)

var _ = ginkgo.Describe("cdc", func() {

	// FIXME: framework will check if all nodes are ready for 3 minutes after each spec, which
	// is time consuming
	f := framework.NewDefaultFramework("cdc")
	f.SkipPrivilegedPSPBinding = true

	var ns string
	var c *util.E2eCli
	var kubeCli *kubernetes.Clientset

	ginkgo.BeforeEach(func() {
		ns = f.Namespace.Name
		e2elog.Logf("Working namespace %s", ns)
		var err error
		kubeCli, err = framework.LoadClientset()
		framework.ExpectNoError(err, "Expected to load kubernetes clientset.")
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

		err = framework.WaitForStatefulSetReplicasReady(mysql.Sts.Name, mysql.Sts.Namespace, kubeCli, 10*time.Second, 5*time.Minute)
		framework.ExpectNoError(err, "Expected mysql ready.")

		cc, err := c.CDC.ApplyCDC(&cdc.CDCSpec{
			Namespace: ns,
			Name:      name,
			Resources: fixture.Small,
			Replicas:  3,
			Source:    tc,
		})
		framework.ExpectNoError(err, "Expected to deploy CDC.")

		err = framework.WaitForStatefulSetReplicasReady(cc.Sts.Name, cc.Sts.Namespace, kubeCli, 10*time.Second, 5*time.Minute)
		framework.ExpectNoError(err, "Expected CDC ready.")

		_ = &cdc.CDCJob{
			CDC:     cc,
			SinkURI: mysql.URI(),
		}
		// start CDC has to run in env that has cdc binary installed and could access PD address, run it manually is a more feasible idea now
		//err = c.CDC.StartJob(&cdc.CDCJob{
		//	CDC:     cc,
		//	SinkURI: mysql.URI(),
		//})
		framework.ExpectNoError(err, "Expected to start a CDC job.")

		// TODO: able to describe describe chaos in BDD-style
		podKill := &chaosv1alpha1.PodChaos{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
			},
			Spec: chaosv1alpha1.PodChaosSpec{
				Selector: chaosv1alpha1.SelectorSpec{
					// Randomly kill in namespace
					Namespaces:     []string{ns},
					LabelSelectors: map[string]string{"app.kubernetes.io/component": "tikv"},
				},
				Scheduler: chaosv1alpha1.SchedulerSpec{
					Cron: "@every 1m",
				},
				Action: chaosv1alpha1.PodKillAction,
				Mode:   chaosv1alpha1.OnePodMode,
			},
		}
		err = c.Chaos.ApplyPodChaos(podKill)
		framework.ExpectNoError(err, "Expected to apply pod chaos.")

		// TODO: write data to TiDB
		tidbService, err := c.TiDB.GetTiDBService(tc)
		framework.ExpectNoError(err, "Expected to get tidb service")
		sourceAddr := putil.GenTiDBServiceAddress(tidbService)

		targetAddr := putil.GenMysqlServiceAddress(mysql.Svc)

		err = workload(sourceAddr, targetAddr, 10, "test", fixture.E2eContext.TimeLimit)
		framework.ExpectNoError(err, "Expected to run workload")
	})
})

func workload(sourceAddr string, targetAddr string, concurrency int, dbName string, timeLimit time.Duration) error {
	executorOption := &executor.Option{
		Clear:  false,
		Stable: true,
		Log:    "/tmp/cdc-log",
	}

	coreOption := &core.Option{
		Concurrency: concurrency,
		Stable:      true,
		Mode:        "binlog",
	}

	exec, err := core.NewABTest(genDbDsn(sourceAddr, dbName), genDbDsn(targetAddr, dbName), coreOption, executorOption)
	if err != nil {
		return err
	}

	if err := exec.PrintSchema(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), timeLimit)
	defer cancel()

	return exec.Start(ctx)
}

func genDbDsn(addr string, dbName string) string {
	return fmt.Sprintf("root:@tcp(%s)/%s", addr, dbName)
}
