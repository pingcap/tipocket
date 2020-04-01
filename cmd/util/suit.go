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

package util

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"
	"github.com/pingcap/tipocket/pkg/loki"
	"github.com/pingcap/tipocket/pkg/nemesis"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/verify"
)

// Version information.
var (
	BuildTS   = "None"
	BuildHash = "None"
)

// PrintInfo prints the octopus version information
func PrintInfo() {
	fmt.Println("Git Commit Hash:", BuildHash)
	fmt.Println("UTC Build Time: ", BuildTS)
}

// Suit is a basic chaos testing suit with configurations to run chaos.
type Suit struct {
	*control.Config
	// Provisioner deploy the SUT cluster
	clusterTypes.Provisioner
	// ClientCreator creates client
	core.ClientCreator
	// NemesisGens saves NemesisGenerator
	NemesisGens []core.NemesisGenerator
	// ClientRequestGen
	ClientRequestGen ClientLoopFunc
	// perform service quality checking
	VerifySuit verify.Suit
	// cluster definition
	ClusterDefs interface{}
}

// Run runs the suit.
func (suit *Suit) Run(ctx context.Context) {
	var (
		err         error
		clusterSpec = clusterTypes.ClusterSpecs{
			Defs:        suit.ClusterDefs,
			NemesisGens: nemesisGeneratorNames(suit.NemesisGens),
		}
	)
	sctx, cancel := context.WithCancel(ctx)
	suit.Config.Nodes, suit.Config.ClientNodes, err = suit.Provisioner.SetUp(sctx, clusterSpec)
	if err != nil {
		log.Fatalf("deploy a cluster failed, err: %s", err)
	}
	log.Printf("deploy cluster success, node:%+v, client node:%+v", suit.Config.Nodes, suit.Config.ClientNodes)
	if len(suit.Config.ClientNodes) == 0 {
		log.Panic("no client nodes exist")
	}
	if suit.Config.ClientCount == 0 {
		log.Panic("suit.Config.ClientCount is required")
	}
	// fill clientNodes
	retClientCount := len(suit.Config.ClientNodes)
	for len(suit.Config.ClientNodes) < suit.Config.ClientCount {
		suit.Config.ClientNodes = append(suit.Config.ClientNodes,
			suit.Config.ClientNodes[rand.Intn(retClientCount)])
	}
	c := control.NewController(
		sctx,
		suit.Config,
		suit.ClientCreator,
		suit.NemesisGens,
		suit.ClientRequestGen,
		suit.VerifySuit,
		loki.NewClient(fixture.Context.LokiAddress,
			fixture.Context.LokiUsername, fixture.Context.LokiPassword),
	)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		<-sigs
		c.Close()
		cancel()
	}()

	c.Run()

	log.Printf("tear down cluster...")
	if err := suit.Provisioner.TearDown(context.TODO(), clusterSpec); err != nil {
		log.Printf("Provisioner tear down failed: %+v", err)
	}
}

// OnClientLoop sends client requests in a loop,
// client applies a proc id as it's identifier and if the response is some kinds of `Unknown` type,
// it will change a proc id on the next loop.
// Each request costs a requestCount, and loop finishes after requestCount is used up or the `ctx` has been done.
func OnClientLoop(
	ctx context.Context,
	client core.Client,
	node clusterTypes.ClientNode,
	proc *int64,
	requestCount *int64,
	recorder *history.Recorder,
) {
	log.Printf("begin to run command on node %s", node)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	procID := atomic.AddInt64(proc, 1)
	for atomic.AddInt64(requestCount, -1) >= 0 {
		request := client.NextRequest()

		if err := recorder.RecordRequest(procID, request); err != nil {
			log.Fatalf("record request %v failed %v", request, err)
		}

		log.Printf("%s: call %+v", node, request)
		response := client.Invoke(ctx, node, request)
		log.Printf("%s: return %+v", node, response)
		isUnknown := true
		if v, ok := response.(core.UnknownResponse); ok {
			isUnknown = v.IsUnknown()
		}

		if err := recorder.RecordResponse(procID, response); err != nil {
			log.Fatalf("record response %v failed %v", response, err)
		}

		// If Unknown, we need to use another process ID.
		if isUnknown {
			procID = atomic.AddInt64(proc, 1)
		}

		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// ClientLoopFunc defines ClientLoop func
type ClientLoopFunc func(ctx context.Context,
	client core.Client,
	node clusterTypes.ClientNode,
	proc *int64,
	requestCount *int64,
	recorder *history.Recorder)

// BuildClientLoopThrottle receives a duration and build a ClientLoopFunc that sends a request every `duration` time
func BuildClientLoopThrottle(duration time.Duration) ClientLoopFunc {
	return func(ctx context.Context,
		client core.Client,
		node clusterTypes.ClientNode,
		proc *int64,
		requestCount *int64,
		recorder *history.Recorder) {
		log.Printf("begin to run command on node %s", node)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		token := make(chan struct{})
		go func() {
			time.Sleep(time.Duration(rand.Int63n(int64(duration))))
			ticker := time.NewTicker(duration)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					close(token)
					return
				case _ = <-ticker.C:
					token <- struct{}{}
				}
			}
		}()

		procID := atomic.AddInt64(proc, 1)
		for atomic.AddInt64(requestCount, -1) >= 0 {
			if _, ok := <-token; !ok {
				return
			}
			request := client.NextRequest()
			if err := recorder.RecordRequest(procID, request); err != nil {
				log.Fatalf("record request %v failed %v", request, err)
			}

			log.Printf("[%d] %s: call %+v", procID, node.String(), request)
			response := client.Invoke(ctx, node, request)
			log.Printf("[%d] %s: return %+v", procID, node.String(), response)
			isUnknown := true
			if v, ok := response.(core.UnknownResponse); ok {
				isUnknown = v.IsUnknown()
			}

			if err := recorder.RecordResponse(procID, response); err != nil {
				log.Fatalf("record response %v failed %v", response, err)
			}

			// If Unknown, we need to use another process ID.
			if isUnknown {
				procID = atomic.AddInt64(proc, 1)
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}
}

// ParseNemesisGenerators parses NemesisGenerator from string literal
func ParseNemesisGenerators(names string) (nemesisGens []core.NemesisGenerator) {
	for _, name := range strings.Split(names, ",") {
		name := strings.TrimSpace(name)
		if len(name) == 0 {
			continue
		}
		nemesisGens = append(nemesisGens, parseNemesisGenerator(name))
	}
	return
}

func parseNemesisGenerator(name string) (g core.NemesisGenerator) {
	switch name {
	case "random_kill", "all_kill", "minor_kill", "major_kill",
		"kill_tikv_1node_5min", "kill_tikv_2node_5min",
		"kill_pd_leader_5min", "kill_pd_nonleader_5min":
		g = nemesis.NewKillGenerator(name)
	case "short_kill_tikv_1node", "short_kill_pd_leader", "short_kill_tiflash_1node":
		g = nemesis.NewContainerKillGenerator(name)
	case "random_drop", "all_drop", "minor_drop", "major_drop":
		log.Panic("Unimplemented")
	case "small_skews", "subcritical_skews", "critical_skews", "big_skews", "huge_skews", "strobe_skews":
		g = nemesis.NewTimeChaos(name)
	case "partition_one":
		g = nemesis.NewNetworkPartitionGenerator(name)
	case "loss", "delay", "duplicate", "corrupt":
		g = nemesis.NewNetemChaos(name)
	case "pod_kill":
		g = nemesis.NewPodKillGenerator(name)
	case "shuffle-leader-scheduler", "shuffle-region-scheduler", "random-merge-scheduler":
		g = nemesis.NewSchedulerGenerator(name)
	case "scaling":
		g = nemesis.NewScalingGenerator(name)
	// TODO: Change that name
	case "leader-shuffle":
		g = nemesis.NewLeaderShuffleGenerator(name)
	case "delay_tikv", "delay_pd", "delay_tiflash", "errno_tikv", "errno_pd",
		"errno_tiflash", "mixed_tikv", "mixed_pd", "mixed_tiflash", "readerr_tikv", "readerr_tiflash":
		g = nemesis.NewIOChaosGenerator(name)
	default:
		log.Fatalf("invalid nemesis generator %s", name)
	}
	return
}

func nemesisGeneratorNames(gens []core.NemesisGenerator) (names []string) {
	for _, gen := range gens {
		names = append(names, gen.Name())
	}
	return
}
