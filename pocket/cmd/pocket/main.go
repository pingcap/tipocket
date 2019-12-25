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

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	_ "github.com/pingcap/tidb/types/parser_driver"
	"github.com/pingcap/tipocket/pocket/config"
	"github.com/pingcap/tipocket/pocket/core"
	"github.com/pingcap/tipocket/pocket/util"
)

// flag names
var (
	// global
	nmPrintVersion = "V"
	nmConfigPath = "config"
	// connection
	nmMode = "mode"
	nmDsn1 = "dsn1"
	nmDsn2 = "dsn2"
	// common options
	nmClearDB = "clear"
	nmStable = "stable"
	nmReproduce = "reproduce"
	nmConcurrency = "concurrency"
	nmPath = "log"
	nmDuration = "duration"
	// misc
	nmPrintSchema = "schema"
)

// arguments
var (
	// global
	printVersion = flag.Bool(nmPrintVersion, false, "print version")
	configPath   = flag.String(nmConfigPath, "", "config file path")
	// connection
	mode = flag.String(nmMode, "", "test mode, can be single, abtest and binlog")
	dsn1 = flag.String(nmDsn1, "", "dsn1")
	dsn2 = flag.String(nmDsn2, "", "dsn2")
	// common options
	clearDB     = flag.Bool(nmClearDB, false, "drop all tables in target database and then start testing")
	stable      = flag.Bool(nmStable, false, "generate stable SQL without random or other env related expression")
	reproduce   = flag.Bool(nmReproduce, false, "reproduce from log")
	concurrency = flag.Int(nmConcurrency, 3, "test concurrency")
	path        = flag.String(nmPath, "", "path")
	duration    = flag.Duration(nmDuration, time.Hour, "the duration time to run test")
	// misc
	printSchema = flag.Bool(nmPrintSchema, false, "print schema and exit")
)

// variables
var (
	cfg = config.Init()
)

func main() {
	flag.Parse()
	if *printVersion {
		util.PrintInfo()
		return
	}

	var (
		c   *core.Core
		err error
	)

	if err := loadConfig(); err != nil {
		log.Fatalf("load config error %+v", errors.ErrorStack(err))
	}

	if err != nil {
		log.Fatalf("create executor error %+v", errors.ErrorStack(err))
	}

	ctx, cancel := context.WithTimeout(context.TODO(), cfg.Options.Duration.Duration)
	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc,
			os.Kill,
			os.Interrupt,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT)

		log.Infof("Got signal %d to exit.", <-sc)
		cancel()
		os.Exit(0)
	}()

	c = core.New(cfg)
	if err := c.Start(ctx); err != nil {
		log.Fatalf("start error: %+v", errors.ErrorStack(err))
	}
}

func loadConfig() error {
	actualFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		actualFlags[f.Name] = true
	})

	if actualFlags[nmConfigPath] {
		if err := cfg.Load(*configPath); err != nil {
			return errors.Trace(err)
		}
	}

	// global config
	if actualFlags[nmMode] {
		cfg.Mode = *mode
	}
	if actualFlags[nmDsn1] {
		cfg.DSN1 = *dsn1
	}
	if actualFlags[nmDsn2] {
		cfg.DSN2 = *dsn2
	}
	// options
	if actualFlags[nmClearDB] {
		cfg.Options.ClearDB = *clearDB
	}
	if actualFlags[nmStable] {
		cfg.Options.Stable = *stable
	}
	if actualFlags[nmReproduce] {
		cfg.Options.Reproduce = *reproduce
	}
	if actualFlags[nmConcurrency] {
		cfg.Options.Concurrency = *concurrency
	}
	if actualFlags[nmPath] {
		cfg.Options.Path = *path
	}
	if actualFlags[nmDuration] {
		cfg.Options.Duration.Duration = *duration
	}

	return nil
}
