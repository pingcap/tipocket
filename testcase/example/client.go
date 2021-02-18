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

package example

import (
	"context"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

// CaseCreator is a creator of test client
type CaseCreator struct{}

// Create creates a test client
func (c CaseCreator) Create(node cluster.ClientNode) core.Client {
	return &Client{}
}

// Client defines how our test case works
type Client struct{}

// SetUp implements the core.Client interface.
func (c *Client) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	return nil
}

// TearDown implements the core.Client interface.
func (c *Client) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

// Start implements the core.StandardClientExtensions interface.
func (c *Client) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	return nil
}
