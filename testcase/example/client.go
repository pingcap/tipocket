// Copyright 2021 PingCAP, Inc.
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

package testcase

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/util"
)

// CaseCreator is a creator of test client
type CaseCreator struct{}

// Create creates a test client
func (c CaseCreator) Create(node cluster.ClientNode) core.Client {
	return &Client{}
}

// Client defines how our test case works
type Client struct {
	db *sql.DB
}

// SetUp implements the core.Client interface.
func (c *Client) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	log.Info("start to setup client...")
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port)
	util.SetMySQLProxy(fixture.Context.MySQLProxy)
	db, err := util.OpenDB(dsn, 1)
	if err != nil {
		log.Fatalf("open db error: %v", err)
	}

	util.MustExec(db, "drop table if exists t")
	util.MustExec(db, "create table t(id int)")
	util.MustExec(db, "insert into t(id) values(1)")

	c.db = db
	return nil
}

// TearDown implements the core.Client interface.
func (c *Client) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	util.MustExec(c.db, "drop table t")
	return nil
}

// Start implements the core.StandardClientExtensions interface.
func (c *Client) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	rows, err := c.db.QueryContext(ctx, "select count(*) from t where id = 1")
	if err != nil {
		return err
	}
	defer rows.Close()
	var count = 0
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return err
		}
	}
	if count != 1 {
		return fmt.Errorf("expect 1, got %d", count)
	}
	log.Info("everything is ok!")
	return nil
}
