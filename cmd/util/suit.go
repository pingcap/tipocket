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
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/chaos-mesh/matrix/api"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"
	"github.com/pingcap/tipocket/pkg/loki"
	"github.com/pingcap/tipocket/pkg/nemesis"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
	"github.com/pingcap/tipocket/pkg/verify"
	"github.com/pingcap/tipocket/util"
)

// Suit is a basic chaos testing suit with configurations to run chaos.
type Suit struct {
	*control.Config
	// Provider deploy the SUT cluster
	cluster.Provider
	// ClientCreator creates client
	core.ClientCreator
	// NemesisGens saves NemesisGenerator
	NemesisGens []core.NemesisGenerator
	// ClientRequestGen
	ClientRequestGen ClientLoopFunc
	// perform service quality checking
	VerifySuit verify.Suit
	// cluster definition
	ClusterDefs cluster.Cluster
	// Plugins
	Plugins []control.Plugin
}

// Run runs the suit.
func (suit *Suit) Run(ctx context.Context) {
	var (
		err         error
		clusterSpec = cluster.Specs{
			Cluster:   suit.ClusterDefs,
			Namespace: fixture.Context.Namespace,
		}
	)
	sctx, cancel := context.WithCancel(ctx)
	// get the time before creating the tidb cluster
	// note this is just a approximate value
	startTime := time.Now()

	// Apply Matrix config
	var matrixSqlStmts []string
	tidbConfig := fixture.Context.TiDBClusterConfig
	if tidbConfig.MatrixConfig.MatrixConfigFile != "" {
		matrixCtx, err := ioutil.TempDir("", "matrix")
		if err != nil {
			log.Warn(fmt.Sprintf("Failed to create Matrix context folder: `%s`, skip Matrix.", err.Error()))
		} else {
			if !tidbConfig.MatrixConfig.NoCleanup {
				defer os.RemoveAll(matrixCtx)
			}

			err = api.Gen(tidbConfig.MatrixConfig.MatrixConfigFile, matrixCtx, 0)
			if err != nil {
				log.Warn("Matrix generation failed, skip Matrix.")
			} else {
				checkAndOverrideConfig := func(matrixConfig string, realConfig *string) error {
					if matrixConfig != "" {
						joinedMatrixConfig := path.Join(matrixCtx, matrixConfig)
						if util.IsFileExist(joinedMatrixConfig) {
							if *realConfig != "" {
								return errors.New(fmt.Sprintf("config file already specified: `%s`", *realConfig))
							}
							*realConfig = joinedMatrixConfig
						} else {
							return errors.New(fmt.Sprintf("`%s` not exists in Matrix output", matrixConfig))
						}
					}
					return nil
				}

				if err = checkAndOverrideConfig(tidbConfig.MatrixConfig.MatrixTiDBConfig, &tidbConfig.TiDBConfig); err != nil {
					log.Warn(fmt.Sprintf("Error applying Matrix's TiDB config, skipped: %s", err.Error()))
				}
				if err = checkAndOverrideConfig(tidbConfig.MatrixConfig.MatrixTiKVConfig, &tidbConfig.TiKVConfig); err != nil {
					log.Warn(fmt.Sprintf("Error applying Matrix's TiKV config, skipped: %s", err.Error()))
				}
				if err = checkAndOverrideConfig(tidbConfig.MatrixConfig.MatrixPDConfig, &tidbConfig.PDConfig); err != nil {
					log.Warn(fmt.Sprintf("Error applying Matrix's PD config, skipped: %s", err.Error()))
				}

				for _, sqlFile := range tidbConfig.MatrixConfig.MatrixSQLConfig {
					sqlFile = path.Join(matrixCtx, sqlFile)
					var b []byte
					b, err = ioutil.ReadFile(sqlFile)
					if err != nil {
						log.Warn(fmt.Sprintf("Error loading from Matrix: %s", err.Error()))
						matrixSqlStmts = nil
						break
					}
					matrixSqlStmts = append(matrixSqlStmts, fmt.Sprint(b))
				}
			}
		}
	}

	suit.Config.Nodes, suit.Config.ClientNodes, err = suit.Provider.SetUp(sctx, clusterSpec)

	for _, node := range suit.ClientNodes {
		if node.Component == cluster.TiDB {
			dsn := fmt.Sprintf("root@tcp(%s:%d)", node.IP, node.Port)
			db, err := util.OpenDB(dsn, 1)
			if err != nil {
				panic("")
			}
			for _, stmt := range matrixSqlStmts {
				_, err = db.Exec(stmt)
				if err != nil {
					panic(err)
				}
			}
		}
	}

	if err != nil {
		// we can release resources safely in this case.
		_ = suit.Provider.TearDown(context.TODO(), clusterSpec)
		log.Fatalf("deploy a cluster failed, maybe has no enough resources, err: %s", err)
	}
	log.Infof("deploy cluster success, node:%+v, client node:%+v", suit.Config.Nodes, suit.Config.ClientNodes)
	if len(suit.Config.ClientNodes) == 0 {
		log.Fatal("no client nodes exist")
	}
	if suit.Config.ClientCount == 0 {
		log.Fatal("suit.Config.ClientCount is required")
	}
	// fill clientNodes
	retClientCount := len(suit.Config.ClientNodes)
	for len(suit.Config.ClientNodes) < suit.Config.ClientCount {
		suit.Config.ClientNodes = append(suit.Config.ClientNodes,
			suit.Config.ClientNodes[rand.Intn(retClientCount)])
	}

	// set loki client
	var lokiCli *loki.Client
	if fixture.Context.LokiAddress != "" {
		lokiCli = loki.NewClient(startTime, fixture.Context.LokiAddress,
			fixture.Context.LokiUsername, fixture.Context.LokiPassword)
	}

	// set plugins
	suit.setDefaultPlugins()

	c := control.NewController(
		sctx,
		suit.Config,
		suit.ClientCreator,
		core.NewNemesisGenerators(suit.NemesisGens),
		suit.ClientRequestGen,
		suit.VerifySuit,
		suit.Plugins,
		lokiCli,
		fixture.Context.LogPath,
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

	log.Info("tear down cluster...")
	if err := suit.Provider.TearDown(context.TODO(), clusterSpec); err != nil {
		log.Infof("Provider tear down failed: %+v", err)
	}
}

func (suit *Suit) setDefaultPlugins() {
	var defaultPlugins = []control.Plugin{
		&control.PanicCheck{},
		&control.LeakCheck{},
	}
	if len(suit.Plugins) == 0 {
		suit.Plugins = defaultPlugins
	}
}

// ClientLoopFunc defines ClientLoop func
type ClientLoopFunc func(ctx context.Context,
	client core.Client,
	node cluster.ClientNode,
	proc *int64,
	requestCount *int64,
	recorder *history.Recorder)

// OnClientLoop sends client requests in a loop,
// client applies a proc id as it's identifier and if the response is some kinds of `Unknown` type,
// it will change a proc id on the next loop.
// Each request costs a requestCount, and loop finishes after requestCount is used up or the `ctx` has been done.
func OnClientLoop(
	ctx context.Context,
	client core.Client,
	node cluster.ClientNode,
	proc *int64,
	requestCount *int64,
	recorder *history.Recorder,
) {
	log.Infof("begin to emit requests on node %s", node)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	procID := atomic.AddInt64(proc, 1)
	for atomic.AddInt64(requestCount, -1) >= 0 {
		request := client.NextRequest()

		if err := recorder.RecordRequest(procID, request); err != nil {
			log.Fatalf("record request %v failed %v", request, err)
		}
		if stringer, ok := request.(fmt.Stringer); ok {
			log.Infof("%d %s: call %s", procID, node, stringer.String())
		} else {
			log.Infof("%d %s: call %+v", procID, node, request)
		}
		response := client.Invoke(ctx, node, request)

		if stringer, ok := response.(fmt.Stringer); ok {
			log.Infof("%d %s: return %+v", procID, node, stringer.String())
		} else {
			log.Infof("%d %s: return %+v", procID, node, response)
		}

		v := response.(core.UnknownResponse)
		isUnknown := v.IsUnknown()

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

// BuildClientLoopThrottle receives a duration and build a ClientLoopFunc that sends a request every `duration` time
func BuildClientLoopThrottle(duration time.Duration) ClientLoopFunc {
	return func(ctx context.Context,
		client core.Client,
		node cluster.ClientNode,
		proc *int64,
		requestCount *int64,
		recorder *history.Recorder) {
		log.Infof("begin to run command on node %s", node)

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

			log.Infof("[%d] %s: call %+v", procID, node.String(), request)
			response := client.Invoke(ctx, node, request)
			log.Infof("[%d] %s: return %+v", procID, node.String(), response)

			v := response.(core.UnknownResponse)
			isUnknown := v.IsUnknown()

			if err := recorder.RecordResponse(procID, response); err != nil {
				log.Fatalf("record response %v failed %v", response, err)
			}

			// If Unknown, we need to use another process ID.
			if isUnknown {
				procID = atomic.AddInt64(proc, 1)
				log.Infof("[%d] %s: procID add 1", procID, node.String())
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
		"kill_pd_leader_5min", "kill_pd_nonleader_5min",
		"kill_dm_1node":
		g = nemesis.NewKillGenerator(name)
	case "short_kill_tikv_1node", "short_kill_pd_leader", "short_kill_tiflash_1node":
		g = nemesis.NewContainerKillGenerator(name)
	case "random_drop", "all_drop", "minor_drop", "major_drop":
		log.Fatal("Unimplemented")
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
