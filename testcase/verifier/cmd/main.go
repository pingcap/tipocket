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
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/verify"
	listappend "github.com/pingcap/tipocket/testcase/list-append"
	rwregister "github.com/pingcap/tipocket/testcase/rw-register"
)

var (
	names = flag.String("names", "list-append", "model names, separated by comma")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		cancel()
	}()

	childCtx, cancel := context.WithCancel(ctx)

	go func() {
		for _, name := range strings.Split(*names, ",") {
			s := verify.Suit{}
			switch name {
			case "list-append":
				s.Model, s.Parser, s.Checker = nil, listappend.AppendParser{}, listappend.AppendChecker{}
			case "rw-register":
				s.Model, s.Parser, s.Checker = nil, rwregister.RegisterParser{}, rwregister.RegisterChecker{}
			default:
				log.Printf("%s is not supported", name)
				continue
			}
			s.Verify(fixture.Context.HistoryFile)
		}

		cancel()
	}()

	<-childCtx.Done()
}
