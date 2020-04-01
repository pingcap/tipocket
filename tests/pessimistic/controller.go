package pessimistic

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/tests/pessimistic/hongbao"
	"github.com/pingcap/tipocket/util"
)

// Config is for pessimisticClient
type Config struct {
	PessimisticClientConfig ClientConfig
	HongbaoClientConfig     hongbao.ClientConfig
}

// ClientCreator creates pessimisticClient
type ClientCreator struct {
	Cfg *Config
}

// Create creates case client
func (l ClientCreator) Create(node types.ClientNode) core.Client {
	return &pessimisticClient{
		cfg: l.Cfg,
	}
}

type pessimisticClient struct {
	TxnMode           string
	Concurrency       int
	cfg               *Config
	db                *sql.DB
	HongbaoClient     *hongbao.Client
	PessimisticClient *Client
	randTxnDB         *sql.DB
	hongbaoDB         *sql.DB
}

func (c *pessimisticClient) SetUp(ctx context.Context, nodes []types.ClientNode, idx int) error {
	var (
		err           error
		node          = nodes[idx]
		dsn           = fmt.Sprintf("root@tcp(%s:%d)/", node.IP, node.Port)
		randTxnDBName = c.cfg.PessimisticClientConfig.DBName
		hongbaoDBName = c.cfg.HongbaoClientConfig.DBName
	)
	log.Infof("start to init...")
	db, err := util.OpenDB(dsn, 1)
	if err != nil {
		return errors.Errorf("[%s] create db client error %v", caseName, err)
	}
	initDBStmt := fmt.Sprintf("drop database if exists %s", randTxnDBName)
	if _, err := db.ExecContext(ctx, initDBStmt); err != nil {
		return errors.Errorf("[%s] initialize database %s err: %v", caseName, randTxnDBName, err)
	}
	initDBStmt = fmt.Sprintf("create database %s", randTxnDBName)
	if _, err := db.ExecContext(ctx, initDBStmt); err != nil {
		return errors.Errorf("[%s] initialize database %s err: %v", caseName, randTxnDBName, err)
	}
	initDBStmt = fmt.Sprintf("drop database if exists %s", hongbaoDBName)
	if _, err := db.ExecContext(ctx, initDBStmt); err != nil {
		return errors.Errorf("[%s] initialize database %s err: %v", caseName, hongbaoDBName, err)
	}
	initDBStmt = fmt.Sprintf("create database %s", hongbaoDBName)
	if _, err := db.ExecContext(ctx, initDBStmt); err != nil {
		return errors.Errorf("[%s] initialize database %s err: %v", caseName, hongbaoDBName, err)
	}
	db.Close()

	var (
		randTxnDBDSN       = fmt.Sprintf("%s%s", dsn, randTxnDBName)
		randTxnConcurrency = c.cfg.PessimisticClientConfig.Concurrency
		hongbaoDBDSN       = fmt.Sprintf("%s%s", dsn, hongbaoDBName)
		hongbaoConcurrency = c.cfg.HongbaoClientConfig.Concurrency
	)
	if c.randTxnDB, err = util.OpenDB(randTxnDBDSN, randTxnConcurrency); err != nil {
		return errors.Errorf("[%s] create db client error %v", caseName, err)
	}
	if c.hongbaoDB, err = util.OpenDB(hongbaoDBDSN, hongbaoConcurrency); err != nil {
		return errors.Errorf("[%s] create db client error %v", hongbao.CaseName, err)
	}

	var (
		wg       sync.WaitGroup
		initErrs []error
	)
	wg.Add(2)

	go func() {
		defer wg.Done()
		c.PessimisticClient = NewPessimisticCase(c.cfg.PessimisticClientConfig)
		initErrs = append(initErrs, c.PessimisticClient.Initialize(ctx, c.randTxnDB))
	}()
	go func() {
		defer wg.Done()
		c.HongbaoClient = hongbao.NewHongbaoCase(&c.cfg.HongbaoClientConfig)
		initErrs = append(initErrs, c.HongbaoClient.Initialize(ctx, c.hongbaoDB))
	}()

	wg.Wait()

	for _, err := range initErrs {
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func (c *pessimisticClient) TearDown(ctx context.Context, nodes []types.ClientNode, idx int) error {
	return nil
}

func (c *pessimisticClient) Invoke(ctx context.Context, node types.ClientNode, r interface{}) interface{} {
	panic("implement me")
}

func (c *pessimisticClient) NextRequest() interface{} {
	panic("implement me")
}

func (c *pessimisticClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("implement me")
}

func (c *pessimisticClient) Start(ctx context.Context, cfg interface{}, clientNodes []types.ClientNode) error {
	ch := make(chan error)

	go func() {
		ch <- c.PessimisticClient.Execute(ctx, c.randTxnDB)
	}()

	go func() {
		ch <- c.HongbaoClient.Execute(ctx, c.hongbaoDB)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-ch:
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
