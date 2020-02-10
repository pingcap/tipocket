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

//
//import (
//	"flag"
//	"fmt"
//	"math/rand"
//	"os"
//	"testing"
//	"time"
//
//	"github.com/pingcap/tipocket/pkg/test-infra/pkg/core"
//	"github.com/pingcap/tipocket/pkg/test-infra/pkg/fixture"
//
//	"k8s.io/kubernetes/test/e2e/framework"
//	"k8s.io/kubernetes/test/e2e/framework/config"
//	"k8s.io/kubernetes/test/e2e/framework/viperconfig"
//)
//
//var viperConfig = flag.String("viper-config", "", "The name of a viper config file (https://github.com/spf13/viper#what-is-viper). "+
//	"All e2e command line parameters can also be configured in such a file. May contain a path and may or may not contain the file suffix. "+
//	"The default is to look for an optional file with `e2e` as base name. If a file is specified explicitly, it must be present.")
//
//var (
//	etcdVersion  string
//	etcdReplicas int
//)
//
//// handleFlags sets up all flags and parses the command line.
//func handleFlags() {
//	flags := flag.CommandLine
//
//	config.CopyFlags(config.Flags, flags)
//	framework.RegisterCommonFlags(flags)
//	framework.RegisterClusterFlags(flags)
//	flags.StringVar(&fixture.E2eContext.LocalVolumeStorageClass, "local-storage-class", "local-storage", "Preferred local volume storageclass of the e2e env")
//	flags.DurationVar(&fixture.E2eContext.TimeLimit, "time-limit", 10*time.Minute, "the duration time to run workload")
//	flags.StringVar(&fixture.E2eContext.Nemesis, "nemesis", "", "the nemesis to inject")
//	flags.IntVar(&fixture.E2eContext.Round, "round", 3, "client test request count")
//	flags.IntVar(&fixture.E2eContext.RequestCount, "request-count", 500, "client test request count")
//	flags.StringVar(&fixture.E2eContext.HistoryFile, "history", "./history.log", "history file")
//	flags.StringVar(&etcdVersion, "version", "3.2.26", "etcd version")
//	flags.IntVar(&etcdReplicas, "replicas", 5, "the replicas of etcd to deploy")
//
//	flag.Parse()
//}
//
//func TestMain(m *testing.M) {
//	// Register test flags, then parse flags.
//	handleFlags()
//
//	// Now that we know which Viper config (if any) was chosen,
//	// parse it and update those options which weren't already set via command line flags
//	// (which have higher priority).
//	if err := viperconfig.ViperizeFlags(*viperConfig, "e2e", flag.CommandLine); err != nil {
//		fmt.Fprintln(os.Stderr, err)
//		os.Exit(1)
//	}
//	framework.AfterReadingAllFlags(&framework.TestContext)
//
//	rand.Seed(time.Now().UnixNano())
//
//	os.Exit(m.Run())
//}
//
//func TestSuit(t *testing.T) {
//	core.RunE2ETests(t)
//}
