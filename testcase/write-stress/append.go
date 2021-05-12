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
	"encoding/base64"
	"math/rand"
	"sync"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/util"
)

type appendClient struct {
	baseClient
}

func (c *appendClient) SetUp(ctx context.Context, nodes []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if err := c.baseClient.SetUp(ctx, nodes, clientNodes, idx); err != nil {
		return err
	}
	util.MustExec(c.db, "drop table if exists write_stress")
	util.MustExec(c.db, "create table write_stress(col1 bigint, col2 varchar(256), data longtext, key k(col1, col2))")
	return nil
}

func (c *appendClient) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	var wg sync.WaitGroup
	for i := 0; i < c.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				err := c.runClient(ctx)
				log.Error(err)
			}
		}()
	}

	wg.Wait()
	log.Info("everything is ok!")
	return nil
}

func (c *appendClient) runClient(ctx context.Context) error {
	rng := rand.New(rand.NewSource(rand.Int63()))

	col2 := make([]byte, 192)
	data := make([]byte, 65536)
	for {
		col1 := rng.Int63()
		col2Len := rng.Intn(192)
		_, _ = rng.Read(col2[:col2Len])
		dataLen := rng.Intn(65536)
		_, _ = rng.Read(data[:dataLen])
		_, err := c.db.ExecContext(ctx, "insert into write_stress values (?, ?, ?)", col1,
			base64.StdEncoding.EncodeToString(col2[:col2Len]),
			base64.StdEncoding.EncodeToString(data[:dataLen]))
		if err != nil {
			return errors.Trace(err)
		}
	}
}
