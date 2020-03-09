package pessimistic

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/tests/pessimistic/hongbao"
	"github.com/pingcap/tipocket/util"
)

// Config is for pessimisticClient
type Config struct {
	TxnMode string `toml:"txn_mode"`
	PessimisticCaseConfig
	hongbao.HongbaoCaseConfig
}

// CaseCreator creates pessimisticClient
type CaseCreator struct {
	Cfg *Config
}

// Create creates case client
func (l CaseCreator) Create(node types.ClientNode) core.Client {
	return &pessimisticClient{
		TxnMode: l.Cfg.TxnMode,
		cfg:     l.Cfg,
	}
}

type pessimisticClient struct {
	TxnMode     string
	Concurrency int
	cfg         *Config
	db          *sql.DB
	*hongbao.HongbaoCase
	*PessimisticCase
	randTxnDB *sql.DB
	hongbaoDB *sql.DB
}

func (c *pessimisticClient) SetUp(ctx context.Context, nodes []types.ClientNode, idx int) error {
	var (
		err           error
		node          = nodes[idx]
		dsn           = fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port)
		randTxnDBName = c.cfg.PessimisticCaseConfig.DBName
		hongbaoDBName = c.cfg.HongbaoCaseConfig.DBName
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
		randTxnDBDSN       = fmt.Sprintf("root@tcp(%s)/%s", node.IP, randTxnDBName)
		randTxnConcurrency = c.cfg.PessimisticCaseConfig.Concurrency
		hongbaoDBDSN       = fmt.Sprintf("root@tcp(%s)/%s", node.IP, hongbaoDBName)
		hongbaoConcurrency = c.cfg.HongbaoCaseConfig.Concurrency
	)
	if c.randTxnDB, err = util.OpenDB(randTxnDBDSN, randTxnConcurrency); err != nil {
		return errors.Errorf("[%s] create db client error %v", caseName, err)
	}
	c.PessimisticCase = NewPessimisticCase(c.cfg.PessimisticCaseConfig)
	if err := c.PessimisticCase.Initialize(ctx, c.randTxnDB); err != nil {
		return errors.Errorf("[%s] initial failed %v", caseName, err)
	}

	if c.hongbaoDB, err = util.OpenDB(hongbaoDBDSN, hongbaoConcurrency); err != nil {
		return errors.Errorf("[%s] create db client error %v", hongbao.CaseName, err)
	}
	c.HongbaoCase = hongbao.NewHongbaoCase(&c.cfg.HongbaoCaseConfig)
	if err := c.HongbaoCase.Initialize(ctx, c.hongbaoDB); err != nil {
		return errors.Errorf("[%s] initial failed %v", hongbao.CaseName, err)
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
		ch <- c.PessimisticCase.Execute(ctx, c.randTxnDB)
	}()

	go func() {
		ch <- c.HongbaoCase.Execute(ctx, c.hongbaoDB)
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
