package crossregion

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/pingcap/log"
	pdClient "github.com/tikv/pd/client"
	"go.uber.org/zap"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/util/pdutil"
	"github.com/pingcap/tipocket/util"
)

const stmtDrop = `
drop table if exists t;
`
const stmtCreate = `
create table t (c int)
PARTITION BY RANGE (c) (
	PARTITION p1 VALUES LESS THAN (6),
	PARTITION p2 VALUES LESS THAN (11),
	PARTITION p3 VALUES LESS THAN (16)
);`

const stmtPolicy = `
alter table t alter partition %v
add placement policy
	constraints='["+zone=%v"]'
	role=leader
	replicas=1`

// Config exposes the config
type Config struct {
	TSORequests int
	DBName      string
	//TODO: support configure DCLocationNum instead of fixed 6
	DCLocationNum int
}

// ClientCreator creates crossRegionClient
type ClientCreator struct {
	Cfg *Config
}

// Create ...
func (l ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &crossRegionClient{
		Config:      l.Cfg,
		initialized: false,
		closed:      false,
	}
}

type crossRegionClient struct {
	*Config
	pdClient     pdClient.Client
	pdHTTPClient *pdutil.Client
	db           *sql.DB
	initialized  bool
	closed       bool
}

// SetUp...
func (c *crossRegionClient) SetUp(ctx context.Context, _ []cluster.Node, cnodes []cluster.ClientNode, idx int) error {
	if c.initialized {
		return nil
	}
	name := cnodes[idx].ClusterName
	namespace := cnodes[idx].Namespace
	pdAddr := buildPDSvcName(name, namespace)
	if c.pdHTTPClient == nil {
		c.pdHTTPClient = pdutil.NewPDClient(http.DefaultClient, "http://"+pdAddr)
	}
	dsn := fmt.Sprintf("root@tcp(%s)/%s", buildTiDBSvcName(name, namespace), c.DBName)
	db, err := util.OpenDB(dsn, 20)
	if err != nil {
		return fmt.Errorf("[on_dup] create db client error %v", err)
	}
	c.db = db
	err = c.setup()
	if c.pdClient == nil {
		c.pdClient, err = pdClient.NewClientWithContext(ctx, []string{pdAddr}, pdClient.SecurityOption{})
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	c.initialized = true
	return nil
}

// TearDown...
func (c *crossRegionClient) TearDown(ctx context.Context, _ []cluster.ClientNode, idx int) error {
	if c.closed {
		return nil
	}
	if c.db != nil {
		c.db.Close()
	}
	if c.pdClient != nil {
		c.pdClient.Close()
	}
	c.closed = true
	return nil
}

// Start...
func (c *crossRegionClient) Start(ctx context.Context, cfg interface{}, cnodes []cluster.ClientNode) error {
	return c.testTSO(ctx)
}

func (c *crossRegionClient) setup() error {
	err := c.setupPD()
	if err != nil {
		return err
	}
	log.Info("PD setup successfully")
	err = c.setupStore()
	if err != nil {
		return err
	}
	log.Info("TiKV setup successfully")
	err = c.setupDB()
	if err != nil {
		return err
	}
	log.Info("DB setup successfully")
	return nil
}

func (c *crossRegionClient) setupPD() error {
	members, err := c.pdHTTPClient.GetMembers()
	if err != nil {
		return err
	}
	if len(members.Members) != 6 {
		return fmt.Errorf("PD member count should be 6")
	}
	err = c.waitLeaderReady()
	if err != nil {
		return err
	}
	log.Info("pd leader ready")
	err = c.waitAllocatorReady([]string{
		"dc-1",
		"dc-2",
		"dc-3",
	})
	if err != nil {
		return err
	}
	log.Info("pd allocator ready")
	return nil
}

func (c *crossRegionClient) setupStore() error {
	stores, err := c.pdHTTPClient.GetStores()
	if err != nil {
		return err
	}
	if stores.Count != 3 {
		return fmt.Errorf("tikv count should be 3")
	}
	i := 0
	for _, store := range stores.Stores {
		i++
		err = c.pdHTTPClient.SetStoreLabels(store.Id, map[string]string{
			"zone": fmt.Sprintf("dc-%v", i),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *crossRegionClient) setupDB() error {
	_, err := c.db.Exec(stmtDrop)
	if err != nil {
		log.Info("setupDB stmtDrop failed", zap.Error(err))
		return err
	}
	_, err = c.db.Exec(stmtCreate)
	if err != nil {
		log.Info("setupDB stmtCreate failed", zap.Error(err))
		return err
	}
	log.Info("setupDB stmtCreate success")
	_, err = c.db.Exec(`set global tidb_enable_alter_placement=1`)
	if err != nil {
		return err
	}
	for i := 1; i <= 3; i++ {
		stmt := fmt.Sprintf(stmtPolicy, fmt.Sprintf("p%v", i), fmt.Sprintf("dc-%v", i))
		_, err = c.db.Exec(stmt)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildPDSvcName(name, namespace string) string {
	return fmt.Sprintf("%s-pd.%s.svc:2379", name, namespace)
}

func buildTiDBSvcName(name, namespace string) string {
	return fmt.Sprintf("%s-tidb.%s.svc:4000", name, namespace)
}
