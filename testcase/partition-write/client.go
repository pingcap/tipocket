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
	"math/rand"
	"strings"
	"sync"

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
	db, err := util.OpenDB(dsn, 16)
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

var alphabet []byte = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

func padString(rng *rand.Rand) string {
	var buf strings.Builder
	buf.Grow(256)
	for i := 0; i < 256; i++ {
		buf.WriteByte(alphabet[rand.Intn(62)])
	}
	return buf.String()
}

// Start implements the core.StandardClientExtensions interface.
func (c *Client) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	if _, err := c.db.ExecContext(ctx, "drop table if exists t"); err != nil {
		return err
	}
	if _, err := c.db.ExecContext(ctx, "create table t (a bigint, pad text, key k(a)) "+
		"SHARD_ROW_ID_BITS = 4 PRE_SPLIT_REGIONS = 4 partition by range (a) ("+
		"partition p0 values less than (1000000), "+
		"partition p1 values less than (2000000), "+
		"partition p2 values less than (3000000))"); err != nil {
		return err
	}

	current := 0
	for ctx.Err() == nil {
		var wg sync.WaitGroup
		for i := 0; i < 512; i++ {
			conn, err := c.db.Conn(ctx)
			if err != nil {
				log.Errorf("failed to get connection: %v", err)
				continue
			}
			wg.Add(1)
			go func(conn *sql.Conn) {
				defer func() {
					conn.Close()
					wg.Done()
				}()
				rng := rand.New(rand.NewSource(rand.Int63()))
				for j := 0; j < 60000; j++ {
					a := rng.Intn(1000000) + current*1000000
					pad := padString(rng)
					if _, err := conn.ExecContext(ctx, "insert into t (a, pad) values (?, ?)", a, pad); err != nil {
						log.Errorf("failed to insert: %v", err)
						if ctx.Err() != nil {
							return
						}
					}
				}
			}(conn)
		}
		wg.Wait()

		go func(p int) {
			toBeCreated := p + 3
			if _, err := c.db.ExecContext(ctx, fmt.Sprintf("alter table t add partition (partition p%d values less than (%d))", toBeCreated, (toBeCreated+1)*1000000)); err != nil {
				log.Errorf("failed to add partition p%d: %v", toBeCreated, err)
			} else {
				log.Infof("succeed to add partition p%d", toBeCreated)
			}
			toBeDeleted := p - 3
			if toBeDeleted >= 0 {
				if _, err := c.db.ExecContext(ctx, fmt.Sprintf("alter table t drop partition p%d", toBeDeleted)); err != nil {
					log.Errorf("failed to drop partition p%d: %v", toBeDeleted, err)
				} else {
					log.Infof("succeed to drop partition p%d", toBeDeleted)
				}
			}

		}(current)

		current++
	}
	return ctx.Err()
}
