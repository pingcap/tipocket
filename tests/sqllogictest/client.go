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

package sqllogictest

import (
	"context"
	"fmt"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

// Config is for sqllogicClient
type Config struct {
	SkipError bool
	TaskCount int
	CaseURL   string
}

type sqllogicClient struct {
	ip   string
	port int32
	*Config
}

// CaseCreator creates sqllogicClient
type CaseCreator struct {
	*Config
}

func (l *CaseCreator) Create(node types.ClientNode) core.Client {
	return &sqllogicClient{
		Config: l.Config,
	}
}

func (c *sqllogicClient) SetUp(ctx context.Context, nodes []types.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}
	node := nodes[idx]
	c.ip = node.IP
	c.port = node.Port
	return nil
}

func (c *sqllogicClient) TearDown(ctx context.Context, nodes []types.ClientNode, idx int) error {
	return nil
}

func (c *sqllogicClient) Invoke(ctx context.Context, node types.ClientNode, r interface{}) interface{} {
	panic("implement me")
}

func (c *sqllogicClient) NextRequest() interface{} {
	panic("implement me")
}

func (c *sqllogicClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("implement me")
}

func (c *sqllogicClient) Start(ctx context.Context, _ interface{}, clientNodes []types.ClientNode) error {
	dbAddr := fmt.Sprintf("%s:%d", c.ip, c.port)
	dbDSN := fmt.Sprintf("root:@tcp(%s)/test", dbAddr)
	db, err := util.OpenDB(dbDSN, len(clientNodes)+1)
	if err != nil {
		log.Fatalf("[sqllogic] initialize error %v", err)
	}
	cfg := &SqllogicTestCaseConfig{
		TestPath:  "./sqllogictest",
		SkipError: c.SkipError,
		Parallel:  c.TaskCount,
		DBName:    "test",
		Host:      dbAddr,
		User:      "root",
		caseURL:   c.CaseURL,
	}
	sqllogic := NewSqllogictest(cfg)
	if err := sqllogic.Initialize(ctx, db); err != nil {
		log.Fatalf("[sqllogic] initialize %s error %v", sqllogic, err)
	}
	if err := sqllogic.Execute(ctx, db); err != nil {
		fmt.Errorf("[sqllogic] execution error %v", err)
	}
	return nil
}
