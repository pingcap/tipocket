package core

import (
	"fmt"
	"os"
	"regexp"
	"sync"
	"path"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/abclient/executor"
	"github.com/pingcap/tipocket/abclient/pkg/logger"
	"github.com/pingcap/tipocket/abclient/pkg/types"
	"github.com/pingcap/tipocket/abclient/connection"
	"github.com/pingcap/tipocket/go-sqlsmith"
)

var (
	dbnameRegex = regexp.MustCompile(`([a-z0-9A-Z_]+)$`)
	schemaConnOption = connection.Option{
		Mute: true,
	}
)

// Executor struct
type Executor struct {
	dsn1            string
	dsn2            string
	coreOpt         *Option
	execOpt         *executor.Option
	executors       []*executor.Executor
	ss              *sqlsmith.SQLSmith
	ch              chan *types.SQL
	logger          *logger.Logger
	dbname          string
	mode            string
	deadlockCh      chan int
	order           *types.Order
	ifLock          bool
	mutex           sync.Mutex
	waitingForCheck bool
	// DSN here is for fetch schema only
	coreConn  *connection.Connection
	coreExec  *executor.Executor
}

// New create Executor
func New(dsn string, coreOpt *Option, execOpt *executor.Option) (*Executor, error) {
	e := Executor{
		dsn1: dsn,
		coreOpt: coreOpt,
		execOpt: execOpt,
		ch: make(chan *types.SQL, 1),
		deadlockCh: make(chan int, 1),
		order: types.NewOrder(),
		mode: "single",
	}

	if coreOpt.Reproduce != "" {
		execOpt.Mute = true
	}
	log.Info("coreOpt.Concurrency is", coreOpt.Concurrency)
	for i := 0; i < coreOpt.Concurrency; i++ {
		opt := execOpt.Clone()
		opt.ID = i + 1
		opt.LogSuffix = fmt.Sprintf("-%d", i + 1)
		exec, err := executor.New(e.dsn1, opt)
		if err != nil {
			return nil, errors.Trace(err)
		}
		e.executors = append(e.executors, exec)
	}
	return e.init()
}

// NewABTest create abtest Executor
func NewABTest(dsn1, dsn2 string, coreOpt *Option, execOpt *executor.Option) (*Executor, error) {
	e := Executor{
		dsn1: dsn1,
		dsn2: dsn2,
		coreOpt: coreOpt,
		execOpt: execOpt,
		ch: make(chan *types.SQL, 1),
		deadlockCh: make(chan int, 1),
		order: types.NewOrder(),
		mode: "abtest",
	}
	if coreOpt.Mode != "" {
		e.mode = coreOpt.Mode
	}

	if coreOpt.Reproduce != "" {
		execOpt.Mute = true
	}
	log.Info("coreOpt.Concurrency is", coreOpt.Concurrency)
	for i := 0; i < coreOpt.Concurrency; i++ {
		opt := execOpt.Clone()
		opt.ID = i + 1
		opt.LogSuffix = fmt.Sprintf("-%d", i + 1)
		var (
			exec *executor.Executor
			err error
		)
		switch e.mode {
		case "abtest":
			exec, err = executor.NewABTest(e.dsn1, e.dsn2, opt)
		case "binlog":
			exec, err = executor.New(e.dsn1, opt)
		}
		if err != nil {
			return nil, errors.Trace(err)
		}
		e.executors = append(e.executors, exec)
	}
	return e.init()
}

func (e *Executor) init() (*Executor, error) {
	// parse dbname
	dbname := dbnameRegex.FindString(e.dsn1)
	if dbname == "" {
		return nil, errors.NotFoundf("empty dbname in dsn")
	}
	e.dbname = dbname
	// init logger
	l, err := logger.New("", false)
	if err != nil {
		return nil, errors.Trace(err)
	}
	e.logger = l
	// init schema exec
	var exec *executor.Executor
	switch e.mode {
	case "single":
		exec, err = executor.New(e.dsn1, e.execOpt.Clone())
	case "abtest":
		exec, err = executor.NewABTest(e.dsn1, e.dsn2, e.execOpt.Clone())
	case "binlog":
		exec, err = executor.New(e.dsn1, e.execOpt.Clone())
	default:
		panic("unhandled mode")
	}
	if err != nil {
		return nil, errors.Trace(err)
	}
	e.coreExec = exec
	e.coreConn = exec.GetConn()
	return e, nil
}

// Start test
func (e *Executor) Start() error {
	if err := e.mustExec(); err != nil {
		return errors.Trace(err)
	}
	if err := e.reloadSchema(); err != nil {
		return errors.Trace(err)
	}
	for _, executor := range e.executors {
		go executor.Start()
	}
	if e.coreOpt.Reproduce != "" {
		l, err := logger.New(path.Join(e.coreOpt.Reproduce, "reproduce.log"), false)
		if err != nil {
			log.Fatal(err)
		}
		e.logger = l
		go e.reproduce()
	} else {
		go e.watchDeadLock()
		go e.smithGenerate()
		go e.startHandler()
		switch e.mode {
		case "abtest":
			go e.startABTestDataCompare()
		case "binlog":
			go e.startBinlogTestDataCompare()
		}
	}
	return nil
}

// Stop test
func (e *Executor) Stop(msg string) {
	log.Infof("[STOP] message: %s\n", msg)
	os.Exit(0)
}
