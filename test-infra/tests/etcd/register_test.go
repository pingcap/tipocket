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
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"go.etcd.io/etcd/clientv3"

	"github.com/anishathalye/porcupine"
	"github.com/pingcap/tipocket/test-infra/pkg/core"
	"github.com/pingcap/tipocket/test-infra/pkg/model"
)

var (
	register = "acc"
)

type registerClient struct {
	db *clientv3.Client
	r  *rand.Rand
}

func (c *registerClient) SetUp(ctx context.Context, nodes []string, node string) error {
	c.r = rand.New(rand.NewSource(time.Now().UnixNano()))
	db, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("%s:2379", node)},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}

	c.db = db

	// Do SetUp in the first node
	if node != nodes[0] {
		return nil
	}

	log.Printf("begin to initial register on node %s", node)

	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	db.Put(pctx, register, "0")
	cancel()

	return nil
}

func (c *registerClient) TearDown(ctx context.Context, nodes []string, node string) error {
	return c.db.Close()
}

func (c *registerClient) invokeRead(ctx context.Context, r model.RegisterRequest) model.RegisterResponse {
	gctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	val, err := c.db.Get(gctx, register)
	cancel()
	if err != nil {
		return model.RegisterResponse{Unknown: true}
	}

	if len(val.Kvs) < 0 {
		panic("get empty value")
	}

	v, err := strconv.ParseInt(string(val.Kvs[0].Value), 10, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid value: %s", val.Kvs[0].Value))
	}
	return model.RegisterResponse{Value: int(v)}
}

func (c *registerClient) Invoke(ctx context.Context, node string, r interface{}) interface{} {
	arg := r.(model.RegisterRequest)
	if arg.Op == model.RegisterRead {
		return c.invokeRead(ctx, arg)
	}

	val := fmt.Sprintf("%d", arg.Value)
	pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	_, err := c.db.Put(pctx, register, val)
	cancel()
	if err != nil {
		return model.RegisterResponse{Unknown: true}
	}
	return model.RegisterResponse{}
}

func (c *registerClient) NextRequest() interface{} {
	r := model.RegisterRequest{
		Op: c.r.Intn(2) == 1,
	}
	if r.Op == model.RegisterRead {
		return r
	}

	r.Value = int(c.r.Int63())
	return r
}

// DumpState the database state(also the model's state)
func (c *registerClient) DumpState(ctx context.Context) (interface{}, error) {
	gctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	val, err := c.db.Get(gctx, register)
	cancel()
	if err != nil {
		return nil, err
	}
	if len(val.Kvs) < 0 {
		return nil, fmt.Errorf("get empty value, key %s", register)
	}

	v, err := strconv.ParseInt(string(val.Kvs[0].Value), 10, 64)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func newRegisterEvent(v interface{}, id uint) porcupine.Event {
	if _, ok := v.(model.RegisterRequest); ok {
		return porcupine.Event{Kind: porcupine.CallEvent, Value: v, Id: id}
	}

	return porcupine.Event{Kind: porcupine.ReturnEvent, Value: v, Id: id}
}

// RegisterClientCreator creates a register test client for rawkv.
type RegisterClientCreator struct {
}

// Create creates a client.
func (RegisterClientCreator) Create(node string) core.Client {
	return &registerClient{}
}
