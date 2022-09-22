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
type CaseCreator struct {
	CaseName    string
	Concurrency int
	Tables      int
	PadLength   int
	DropTable   bool
}

// Create creates a test client
func (c CaseCreator) Create(node cluster.ClientNode) core.Client {
	base := baseClient{
		concurrency: c.Concurrency,
		tables:      c.Tables,
		padLength:   c.PadLength,
		dropTable:   c.DropTable,
	}
	switch c.CaseName {
	case "uniform":
		return &uniformClient{base}
	case "append":
		return &appendClient{base}
	}
	return nil
}

type baseClient struct {
	db          *sql.DB
	concurrency int
	tables      int
	padLength   int
	dropTable   bool
}

func (c *baseClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	log.Info("start to setup client...")
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port)
	util.SetMySQLProxy(fixture.Context.MySQLProxy)
	db, err := util.OpenDB(dsn, c.concurrency)
	if err != nil {
		log.Fatalf("open db error: %v", err)
	}

	c.db = db
	return nil
}

func (c *baseClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}
