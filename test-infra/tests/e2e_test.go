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

package e2e

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	// Never, ever remove the line with "/ginkgo". Without it,
	// the ginkgo test runner will not detect that this
	// directory contains a Ginkgo test suite.
	// See https://github.com/kubernetes/kubernetes/issues/74827
	// "github.com/onsi/ginkgo"

	"github.com/pingcap/tipocket/test-infra/pkg/fixture"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/kubernetes/test/e2e/framework/config"
	"k8s.io/kubernetes/test/e2e/framework/testfiles"
	"k8s.io/kubernetes/test/e2e/framework/viperconfig"
	// test sources
	// _ "github.com/pingcap/tipocket/tests/binlog"
	// _ "github.com/pingcap/tipocket/tests/br"
	_ "github.com/pingcap/tipocket/test-infra/tests/cdc"
)

var viperConfig = flag.String("viper-config", "", "The name of a viper config file (https://github.com/spf13/viper#what-is-viper). All e2e command line parameters can also be configured in such a file. May contain a path and may or may not contain the file suffix. The default is to look for an optional file with `e2e` as base name. If a file is specified explicitly, it must be present.")

// handleFlags sets up all flags and parses the command line.
func handleFlags() {
	flags := flag.CommandLine

	config.CopyFlags(config.Flags, flags)
	framework.RegisterCommonFlags(flags)
	framework.RegisterClusterFlags(flags)
	flags.StringVar(&fixture.E2eContext.LocalVolumeStorageClass, "local-storage-class", "local-storage", "Preferred local volume storageclass of the e2e env")
	flags.StringVar(&fixture.E2eContext.RemoteVolumeStorageClass, "remote-storage-class", "rbd", "Preferred local volume storageclass of the e2e env")
	flags.StringVar(&fixture.E2eContext.TiDBVersion, "tidb-version", "v3.0.7", "Default TiDB cluster version in e2e")
	flags.StringVar(&fixture.E2eContext.CDCImage, "cdc-image", "hub.pingcap.net/aylei/cdc:latest", "Default CDC image in e2e")
	flags.StringVar(&fixture.E2eContext.MySQLVersion, "mysql-version", "5.6", "Default CDC image in e2e")
	flags.StringVar(&fixture.E2eContext.DockerRepository, "docker-repo", "pingcap", "Default docker repository in e2e")
	flags.DurationVar(&fixture.E2eContext.TimeLimit, "time-limit", 1*time.Hour, "the duration time to run workload")
	flag.Parse()
}

func TestMain(m *testing.M) {
	// Register test flags, then parse flags.
	handleFlags()

	// Now that we know which Viper config (if any) was chosen,
	// parse it and update those options which weren't already set via command line flags
	// (which have higher priority).
	if err := viperconfig.ViperizeFlags(*viperConfig, "e2e", flag.CommandLine); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	framework.AfterReadingAllFlags(&framework.TestContext)

	// TODO: Deprecating repo-root over time... instead just use gobindata_util.go , see #23987.
	// Right now it is still needed, for example by
	// test/e2e/framework/ingress/ingress_utils.go
	// for providing the optional secret.yaml file and by
	// test/e2e/framework/util.go for cluster/log-dump.
	if framework.TestContext.RepoRoot != "" {
		testfiles.AddFileSource(testfiles.RootFileSource{Root: framework.TestContext.RepoRoot})
	}
	rand.Seed(time.Now().UnixNano())
	os.Exit(m.Run())
}

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}
