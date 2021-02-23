package crossregion

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/pingcap/log"
	pdClient "github.com/tikv/pd/client"
	"go.uber.org/zap"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/util/pdutil"
	util2 "github.com/pingcap/tipocket/testcase/cross-region/pkg/util"
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
	TestTSO         bool
	TSORequestTimes int
	DBName          string
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
	if c.Config.TestTSO && c.pdClient == nil {
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
	if c.TestTSO {
		return c.testTSO(ctx)
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

func (c *crossRegionClient) testTSO(ctx context.Context) error {
	if err := c.requestTSOs(ctx); err != nil {
		return err
	}
	log.Info("start to transfer Leader")
	if err := c.transferLeader(); err != nil {
		return err
	}
	log.Info("leader transfer committed")
	if err := c.waitLeaderReady(); err != nil {
		return err
	}
	log.Info("new leader ready")
	for i := 1; i <= 3; i++ {
		if err := c.transferPDAllocator(fmt.Sprintf("dc-%d", i)); err != nil {
			return err
		}
	}
	log.Info("new allocators ready")
	if err := c.requestGlobalTSOErr(ctx); err != nil {
		return err
	}
	return c.requestTSOs(ctx)
}

func (c *crossRegionClient) requestTSOs(ctx context.Context) error {
	tsoCtx, tsoCancel := context.WithCancel(ctx)
	defer tsoCancel()
	tsoWg := &sync.WaitGroup{}
	tsoWg.Add(4)
	tsoErrCh := make(chan error, 4)
	go c.requestTSO(tsoCtx, "global", tsoWg, tsoErrCh)
	for i := 1; i <= 3; i++ {
		go c.requestTSO(tsoCtx, fmt.Sprintf("dc-%v", i), tsoWg, tsoErrCh)
	}
	tsoWg.Wait()
	if len(tsoErrCh) < 1 {
		log.Info("requestTSOs success")
		return nil
	}
	var tsoErrors []error
	for len(tsoErrCh) > 0 {
		tsoErr := <-tsoErrCh
		tsoErrors = append(tsoErrors, tsoErr)
	}
	return util2.WrapErrors(tsoErrors)
}

func (c *crossRegionClient) requestTSO(ctx context.Context, dcLocation string, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	if c.pdClient != nil {
		for i := 0; i < c.TSORequestTimes; i++ {
			physical, logical, err := c.pdClient.GetLocalTS(ctx, dcLocation)
			if err != nil {
				log.Error("requestTSO failed", zap.String("dc-location", dcLocation), zap.Error(err))
				errCh <- err
				return
			}
			log.Info("request TSO", zap.String("dcLocation", dcLocation),
				zap.Int64("physical", physical),
				zap.Int64("logical", logical))
		}
	}
}

func (c *crossRegionClient) transferPDAllocator(dcLocation string) error {
	members, err := c.pdHTTPClient.GetMembers()
	if err != nil {
		return err
	}
	var names []string
	for _, member := range members.Members {
		if member.DcLocation == dcLocation {
			names = append(names, member.Name)
		}
	}
	allocatorName := ""
	for dcLocation, allocator := range members.TsoAllocatorLeaders {
		if dcLocation == dcLocation {
			allocatorName = allocator.Name
			break
		}
	}
	if allocatorName == "" || len(names) < 2 {
		return fmt.Errorf("dclocation %v don't have enough member", dcLocation)
	}
	transferName := ""
	for _, name := range names {
		if name != allocatorName {
			transferName = name
			break
		}
	}
	if transferName == "" {
		return fmt.Errorf("dclocation %v haven't find transfer pd member", dcLocation)
	}
	err = c.pdHTTPClient.TransferAllocator(transferName, dcLocation)
	if err != nil {
		return err
	}
	log.Info("TransferAllocator committed", zap.String("dclocation", dcLocation),
		zap.String("target-allocator", transferName),
		zap.String("origin-allocator", allocatorName))
	err = c.waitAllocator(transferName, dcLocation)
	if err != nil {
		return err
	}
	log.Info("TransferAllocator finish", zap.String("dclocation", dcLocation),
		zap.String("target-allocator", transferName))
	return nil
}

func (c *crossRegionClient) transferLeader() error {
	members, err := c.pdHTTPClient.GetMembers()
	if err != nil {
		return err
	}
	targetLeader := ""
	for _, member := range members.Members {
		if member.Name != members.Leader.Name {
			err = c.pdHTTPClient.TransferPDLeader(member.Name)
			if err != nil {
				return err
			}
			targetLeader = member.Name
			break
		}
	}
	return c.waitLeader(targetLeader)
}

func (c *crossRegionClient) waitLeaderReady() error {
	return util2.WaitUntil(func() bool {
		members, err := c.pdHTTPClient.GetMembers()
		if err != nil {
			return false
		}
		return members.Leader != nil
	})
}

func (c *crossRegionClient) waitAllocatorReady(dcLocations []string) error {
	return util2.WaitUntil(func() bool {
		members, err := c.pdHTTPClient.GetMembers()
		if err != nil {
			return false
		}
		for _, dclocation := range dcLocations {
			_, ok := members.TsoAllocatorLeaders[dclocation]
			if !ok {
				return false
			}
		}
		return true
	})
}

func (c *crossRegionClient) waitLeader(name string) error {
	return util2.WaitUntil(func() bool {
		mems, err := c.pdHTTPClient.GetMembers()
		if err != nil {
			return false
		}
		if mems.Leader == nil || mems.Leader.Name != name {
			return false
		}
		return true
	})
}

func (c *crossRegionClient) waitAllocator(name, dcLocation string) error {
	return util2.WaitUntil(func() bool {
		members, err := c.pdHTTPClient.GetMembers()
		if err != nil {
			return false
		}
		for dc, member := range members.TsoAllocatorLeaders {
			if dcLocation == dc && member.Name == name {
				return true
			}
		}
		return false
	})
}

func (c *crossRegionClient) requestGlobalTSOErr(ctx context.Context) error {
	_, _, err := c.pdClient.GetTS(ctx)
	if err == nil {
		return fmt.Errorf("global tso should return error")
	}
	if !strings.Contains(err.Error(), "mismatch leader id") {
		return fmt.Errorf("global tso should return mismatch leader id error")
	}
	return nil
}

func buildPDSvcName(name, namespace string) string {
	return fmt.Sprintf("%s-pd.%s.svc:2379", name, namespace)
}

func buildTiDBSvcName(name, namespace string) string {
	return fmt.Sprintf("%s-tidb.%s.svc:4000", name, namespace)
}
