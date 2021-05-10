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
	"encoding/base64"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/ngaut/log"

	"github.com/pingcap/errors"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/util"
)

// CaseCreator is a creator of test client
type CaseCreator struct {
	Concurrency int
}

// Create creates a test client
func (c CaseCreator) Create(node cluster.ClientNode) core.Client {
	return &Client{
		concurrency: c.Concurrency,
	}
}

// Client defines how our test case works
type Client struct {
	db          *sql.DB
	concurrency int
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

	util.MustExec(db, "drop table if exists write_stress")
	util.MustExec(db, "create table write_stress(id varchar(40) primary key clustered, col1 bigint, col2 varchar(256), data longtext, key k(col1, col2))")

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
	ctx2, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan error)
	for i := 0; i < 256; i++ {
		go func() {
			err := c.runClient(ctx2)
			log.Error(err)
			ch <- err
		}()
	}

	for i := 0; i < 256; i++ {
		err := <-ch
		if err != nil {
			return err
		}
	}

	log.Info("everything is ok!")
	return nil
}

func (c *Client) runClient(ctx context.Context) error {
	rng := rand.New(rand.NewSource(rand.Int63()))

	var txn *sql.Tx
	defer func() {
		if txn != nil {
			_ = txn.Rollback()
		}
	}()

	col2 := make([]byte, 192)
	data := make([]byte, 65536)
	for {
		txn, err := c.db.BeginTx(ctx, &sql.TxOptions{})
		if err != nil {
			return errors.Trace(err)
		}
		for i := 0; i < 10; i++ {
			uuid := uuid.New().String()
			col1 := rng.Int63()
			col2Len := rng.Intn(192)
			_, _ = rng.Read(col2[:col2Len])
			dataLen := rng.Intn(65536)
			_, _ = rng.Read(data[:dataLen])
			_, err := txn.ExecContext(ctx, "insert into write_stress values (?, ?, ?, ?)", uuid, col1,
				base64.StdEncoding.EncodeToString(col2[:col2Len]),
				base64.StdEncoding.EncodeToString(data[:dataLen]))
			if err != nil {
				return errors.Trace(err)
			}
		}
		err = txn.Commit()
		if err != nil {
			return errors.Trace(err)
		}
	}
}
