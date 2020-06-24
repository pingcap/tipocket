package deadlock

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

// Config is for detectorClient
type Config struct {
	DBName           string
	TableNum         int
	DeadlockInterval time.Duration
	DeadlockTimeout  time.Duration
}

type detectorClient struct {
	*Config
	*singleStatementRollbackCase
	*deadlockCase
	db *sql.DB
}

// ClientCreator creates detectorClient
type ClientCreator struct {
	Cfg *Config
}

// Create ...
func (l ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &detectorClient{
		Config: l.Cfg,
	}
}

func (c *detectorClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port)

	c.deadlockCase = newDeadlockCase(dsn, c.DeadlockInterval, c.DeadlockTimeout)
	if err := c.deadlockCase.initialize(ctx); err != nil {
		return fmt.Errorf("[detector leader change] Initialize deadlock case failed: %+v", err)
	}

	c.singleStatementRollbackCase = newSingleStatementRollbackCase(dsn, c.TableNum, c.DeadlockInterval)
	if err := c.singleStatementRollbackCase.initialize(ctx); err != nil {
		return fmt.Errorf("[detector leader change] Initialize single statement rollback case failed: %+v", err)
	}

	return nil
}

func (c *detectorClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (c *detectorClient) Invoke(ctx context.Context, node cluster.ClientNode, r interface{}) core.UnknownResponse {
	panic("implement me")
}

func (c *detectorClient) NextRequest() interface{} {
	panic("implement me")
}

func (c *detectorClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("implement me")
}

func (c *detectorClient) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	log.Infof("start to test...")
	defer func() {
		log.Infof("test end...")
	}()

	go c.deadlockCase.execute(ctx)
	c.singleStatementRollbackCase.execute(ctx)
	return nil
}
