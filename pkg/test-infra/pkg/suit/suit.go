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

package suit

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/pingcap/tipocket/pkg/test-infra/pkg/control"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/core"
	"github.com/pingcap/tipocket/pkg/test-infra/pkg/verify"
)

// Suit is a basic chaos testing suit with configurations to run chaos.
type Suit struct {
	*control.Config
	core.ClientCreator

	VerifySuit verify.Suit
}

// Run runs the suit.
func (suit *Suit) Run(ctx context.Context, nodes []string) {
	sctx, cancel := context.WithCancel(ctx)

	if len(nodes) != 0 {
		suit.Config.Nodes = nodes
	} else {
		// By default, we run TiKV/TiDB cluster on 5 nodes.
		for i := 1; i <= 5; i++ {
			name := fmt.Sprintf("n%d", i)
			suit.Config.Nodes = append(suit.Config.Nodes, name)
		}
	}

	c := control.NewController(
		sctx,
		suit.Config,
		suit.ClientCreator,
		suit.VerifySuit,
	)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		c.Close()
		cancel()
	}()

	c.Run()
}
