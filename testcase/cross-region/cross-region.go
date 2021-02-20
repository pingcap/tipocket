package crossregion

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"

	"github.com/pingcap/log"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/util/pdutil"
	util2 "github.com/pingcap/tipocket/testcase/cross-region/pkg/util"
	"github.com/pingcap/tipocket/util"
	pdClient "github.com/tikv/pd/client"
	"go.uber.org/zap"
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

type Config struct {
	TestTSO         bool
	TSORequestTimes int
	DBName          string
}

// ClientCreator creates crossRegionClient
type ClientCreator struct {
	Cfg *Config
}

// Create ...
func (l ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &crossRegionClient{
		Config: l.Cfg,
	}
}

type crossRegionClient struct {
	*Config
	pdClient     pdClient.Client
	pdHttpClient *pdutil.Client
	db           *sql.DB
}

func (c *crossRegionClient) SetUp(ctx context.Context, nodes []cluster.Node, cnodes []cluster.ClientNode, idx int) error {
	pdNodePort := int32(0)
	tidbNodePort := int32(0)
	for _, cnode := range cnodes {
		if cnode.Component == cluster.PD {
			pdNodePort = cnode.NodePort
		} else if cnode.Component == cluster.TiDB {
			tidbNodePort = cnode.NodePort
		}
	}
	pdAddr := buildAddr("172.16.4.19", pdNodePort)
	c.pdHttpClient = pdutil.NewPDClient(http.DefaultClient, pdAddr)
	dsn := fmt.Sprintf("root@tcp(%s:%d)/%s", "172.16.4.19", tidbNodePort, c.DBName)
	db, err := util.OpenDB(dsn, 20)
	if err != nil {
		return fmt.Errorf("[on_dup] create db client error %v", err)
	}
	c.db = db
	err = c.setup(cnodes[idx].Component)
	if c.Config.TestTSO {
		c.pdClient, err = pdClient.NewClientWithContext(ctx, []string{pdAddr}, pdClient.SecurityOption{})
		if err != nil {
			return err
		}
	}
	return err
}

func (c *crossRegionClient) TearDown(ctx context.Context, _ []cluster.ClientNode, idx int) error {
	if c.db != nil {
		c.db.Close()
	}
	if c.pdClient != nil {
		c.pdClient.Close()
	}
	if c.pdClient != nil {
		c.pdClient.Close()
	}
	return nil
}

func (c *crossRegionClient) Start(ctx context.Context, cfg interface{}, cnodes []cluster.ClientNode) error {
	if c.TestTSO {
		return c.testTSO(ctx)
	}
	return nil
}

func (c *crossRegionClient) setup(typ cluster.Component) error {
	switch typ {
	case cluster.PD:
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
	case cluster.TiDB:
		err := c.setupDB()
		if err != nil {
			return err
		}
		log.Info("DB setup successfully")
	}
	return nil
}

func (c *crossRegionClient) setupPD() error {
	members, err := c.pdHttpClient.GetMembers()
	if err != nil {
		return err
	}
	if len(members.Members) != 6 {
		return fmt.Errorf("PD member count should be 6")
	}
	i := 0
	for _, member := range members.Members {
		i++
		dcLocation := fmt.Sprintf("dc-%v", i/2+i%2)
		err = c.pdHttpClient.SetMemberDCLocation(member.Name, dcLocation)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *crossRegionClient) setupStore() error {
	stores, err := c.pdHttpClient.GetStores()
	if err != nil {
		return err
	}
	if stores.Count != 3 {
		return fmt.Errorf("tikv count should be 3")
	}
	i := 0
	for _, store := range stores.Stores {
		i++
		err = c.pdHttpClient.SetStoreLabels(store.Id, map[string]string{
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
	err := c.requestTSOs(ctx)
	if err != nil {
		return err
	}
	err = c.TransferLeader()
	if err != nil {
		return err
	}
	for i := 1; i <= 3; i++ {
		err = c.TransferPDAllocator(fmt.Sprintf("dc-%d", i))
		if err != nil {
			return err
		}
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

func (c *crossRegionClient) TransferPDAllocator(dcLocation string) error {
	members, err := c.pdHttpClient.GetMembers()
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
	err = c.pdHttpClient.TransferAllocator(transferName, dcLocation)
	if err != nil {
		return err
	}
	return nil
}

func (c *crossRegionClient) TransferLeader() error {
	members, err := c.pdHttpClient.GetMembers()
	if err != nil {
		return err
	}
	for _, member := range members.Members {
		if member.Name != members.Leader.Name {
			err = c.pdHttpClient.TransferPDLeader(member.Name)
			if err != nil {
				return err
			}
			break
		}
	}
	return fmt.Errorf("failed to transfer pd leader")
}

func (c *crossRegionClient) WaitLeader(name string) error {
	return nil
}

func (c *crossRegionClient) WaitAllocator(name, dclocation string) error {
	return nil
}

func buildPDSvcName(name, namespace string) string {
	return fmt.Sprintf("%s-pd.%s.svc:2379", name, namespace)
}

func buildAddr(addr string, port int32) string {
	return fmt.Sprintf("http://%s:%v", addr, port)
}
