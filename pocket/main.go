package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/pingcap/tipocket/pocket/core"
	"github.com/pingcap/tipocket/pocket/executor"
	"github.com/pingcap/tipocket/pocket/util"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	_ "github.com/pingcap/tidb/types/parser_driver"
)

var (
	printVersion bool
	dsn1        string
	dsn2        string
	printSchema bool
	clearDB     bool
	logPath     string
	reproduce   string
	stable      bool
	concurrency int
	binlog      bool
)

func init() {
	flag.BoolVar(&printVersion, "V", false, "print version")
	flag.BoolVar(&printSchema, "schema", false, "print schema and exit")
	flag.StringVar(&dsn1, "dsn1", "", "dsn1")
	flag.StringVar(&dsn2, "dsn2", "", "dsn2")
	flag.BoolVar(&clearDB, "clear", false, "drop all tables in target database and then start testing")
	flag.StringVar(&logPath, "log", "", "log path")
	flag.StringVar(&reproduce, "re", "", "reproduce from logs, dir:table, will execute only the given table unless all table")
	flag.BoolVar(&stable, "stable", false, "generate stable SQL without random or other env related expression")
	flag.BoolVar(&binlog, "binlog", false, "enable binlog test which requires 2 dsn points but only write first")
	flag.IntVar(&concurrency, "concurrency", 1, "test concurrency")
}

func main() {
	flag.Parse()
	if printVersion {
		util.PrintInfo()
		os.Exit(0)
	}

	var (
		exec *core.Executor
		err error
		executorOption = executor.Option{}
		coreOption = core.Option{}
	)

	executorOption.Clear = clearDB
	executorOption.Log = logPath
	executorOption.Reproduce = reproduce
	executorOption.Stable = stable
	coreOption.Stable = stable
	coreOption.Concurrency = concurrency
	coreOption.Reproduce = reproduce
	if binlog {
		coreOption.Mode = "binlog"
	}

	if dsn1 == "" {
		log.Fatalf("dsn1 can not be empty")
	} else if dsn2 == "" {
		exec, err = core.New(dsn1, &coreOption, &executorOption)
	} else {
		exec, err = core.NewABTest(dsn1, dsn2, &coreOption, &executorOption)
	}
	if err != nil {
		log.Fatalf("create executor error %v", errors.ErrorStack(err))
	}

	if printSchema {
		if err := exec.PrintSchema(); err != nil {
			log.Fatalf("print schema err %v", errors.ErrorStack(err))
		}
		os.Exit(0)
	}
	if err := exec.Start(); err != nil {
		log.Fatalf("start exec error %v", err)
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	log.Infof("Got signal %d to exit.", <-sc)
}
