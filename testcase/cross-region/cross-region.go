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

// DCLocations stores all the dcLocations' name.
var DCLocations = []string{}

// Config exposes the config
type Config struct {
	Round       int
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
	for i := 0; i < c.Round; i++ {
		log.Info("Start to test TSO", zap.Int("round", i))
		if err := c.testTSO(ctx); err != nil {
			return err
		}
	}
	return nil
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
	err = c.waitAllocatorReady(DCLocations)
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
	if stores.Count != 1 {
		return fmt.Errorf("tikv count should be 1")
	}
	for i, store := range stores.Stores {
		err = c.pdHTTPClient.SetStoreLabels(store.Id, map[string]string{
			"zone": DCLocations[i],
		})
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
