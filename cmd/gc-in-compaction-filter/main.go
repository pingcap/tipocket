// Copyright 2020 PingCAP, Inc.
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

package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/ngaut/log"

	"github.com/tikv/client-go/config"
	"github.com/tikv/client-go/rawkv"
	etcd "go.etcd.io/etcd/clientv3"

	cmd_util "github.com/pingcap/tipocket/cmd/util"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	test_infra "github.com/pingcap/tipocket/pkg/test-infra"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/util"
)

const (
	caseLabel = "gc-in-compaction-filter"
)

type clientCreator struct{}
type client struct {
	pdAddrs  []string
	kvAddrs  []string
	rawKvCli *rawkv.Client
	etcdCli  *etcd.Client
	db       *sql.DB
}

func (c clientCreator) Create(_ cluster.ClientNode) core.Client {
	return &client{}
}

func (c *client) String() string {
	return caseLabel
}

func (c *client) SetUp(ctx context.Context, nodes []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	var err error

	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/", node.IP, node.Port)

	log.Infof("[%s] connnecting to tidb %s...", caseLabel, dsn)
	c.db, err = util.OpenDB(dsn, 1)
	if err != nil {
		log.Fatalf("[%s] create db client error %v", caseLabel, err)
	}

	_, err = c.db.Exec(`update mysql.tidb set VARIABLE_VALUE="1m" where VARIABLE_NAME="tikv_gc_run_interval";`)
	if err != nil {
		log.Fatalf("[%s] update gc interval error %v", caseLabel, err)
	}

	for _, node := range nodes {
		if node.Component == cluster.PD {
			addr := fmt.Sprintf("%s:%d", node.IP, node.Port)
			c.pdAddrs = append(c.pdAddrs, addr)
		} else if node.Component == cluster.TiKV {
			addr := fmt.Sprintf("%s:%d", node.IP, node.Port)
			c.kvAddrs = append(c.kvAddrs, addr)
		}
	}

	c.rawKvCli, err = rawkv.NewClient(context.TODO(), c.pdAddrs, config.Default())
	if err != nil {
		log.Fatalf("[%s] create rawkv client error %v", caseLabel, err)
	}

	c.etcdCli, err = etcd.New(etcd.Config{Endpoints: c.pdAddrs})
	if err != nil {
		log.Fatalf("[%s] create etcd client error %v", caseLabel, err)
	}

	if *waitSafePoint {
		// TiDB needs some time (about 3 minutes) to update a safe point to PD.
		// And TiKV needs more time to load it. So, sleep a while.
		log.Infof("[%s] sleep 5 minutes to wait TiDB updates safe point", caseLabel)
		time.Sleep(time.Second * 300)
	}

	return nil
}

func (c *client) TearDown(_ context.Context, _ []cluster.ClientNode, _ int) error {
	c.rawKvCli.Close()
	c.etcdCli.Close()
	return nil
}

func (c *client) Invoke(_ context.Context, _ cluster.ClientNode, _ interface{}) core.UnknownResponse {
	panic("implement me")

}

func (c *client) NextRequest() interface{} {
	panic("implement me")
}

func (c *client) DumpState(_ context.Context) (interface{}, error) {
	panic("implement me")
}

func (c *client) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	c.testGcStaleVersions()
	c.testGcLatestPutBeforeSafePoint()
	c.testGcLatestStaleDeleteMark(true)
	c.testDynamicConfChange()
	return nil
}

var (
	clusterVersion = flag.String("cluster-version", "5.x", "support values: 4.x / 5.x, default value: 5.x")
	localMode      = flag.Bool("local-mode", false, "use local mode or not")
	waitSafePoint  = flag.Bool("wait-safe-point", false, "wait tidb to update safe point or not")
)

func main() {
	flag.Parse()

	cfg := control.Config{
		Mode:        control.ModeSelfScheduled,
		ClientCount: 1,
		RunTime:     fixture.Context.RunTime,
		RunRound:    1,
	}

	var provider cluster.Provider
	if *localMode {
		provider = cluster.NewLocalClusterProvisioner(
			[]string{"127.0.0.1:50011"}, // TiDBs
			[]string{"127.0.0.1:50001"}, // PDs
			[]string{"127.0.0.1:50003"}, // TiKVs
		)
	} else {
		provider = cluster.NewDefaultClusterProvider()
	}

	suit := cmd_util.Suit{
		Config:        &cfg,
		Provider:      provider,
		ClientCreator: clientCreator{},
		NemesisGens:   cmd_util.ParseNemesisGenerators(fixture.Context.Nemesis),
		VerifySuit:    verify.Suit{},
		ClusterDefs: test_infra.NewDefaultCluster(fixture.Context.Namespace, fixture.Context.Namespace,
			fixture.Context.TiDBClusterConfig),
	}
	suit.Run(context.Background())
}
