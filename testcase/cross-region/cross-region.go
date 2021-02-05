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
	c.pdHttpClient = pdutil.NewPDClient(http.DefaultClient, buildAddr("172.16.4.19", pdNodePort))
	dsn := fmt.Sprintf("root@tcp(%s:%d)/%s", "172.16.4.19", tidbNodePort, c.DBName)
	db, err := util.OpenDB(dsn, 20)
	if err != nil {
		return fmt.Errorf("[on_dup] create db client error %v", err)
	}
	c.db = db
	err = c.setup()
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
	errCh := make(chan error, 1)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go c.requestTSOs(ctx, wg, errCh)
	wg.Wait()
	if len(errCh) < 1 {
		return nil
	}
	var errs []error
	for len(errCh) < 1 {
		err := <-errCh
		errs = append(errs, err)
	}
	return util2.WrapErrors(errs)
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
		return err
	}
	_, err = c.db.Exec(stmtCreate)
	if err != nil {
		return err
	}
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

func (c *crossRegionClient) requestTSOs(ctx context.Context, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	tsoCtx, tsoCancel := context.WithCancel(ctx)
	defer tsoCancel()
	tsoWg := &sync.WaitGroup{}
	tsoWg.Add(4)
	tsoErrCh := make(chan error, 4)
	go c.requestTSO(tsoCtx, "global", tsoWg, errCh)
	for i := 1; i <= 3; i++ {
		go c.requestTSO(tsoCtx, fmt.Sprintf("dc-%v", i), wg, tsoErrCh)
	}
	tsoWg.Wait()
	if len(tsoErrCh) < 1 {
		return
	}
	var tsoErrors []error
	for len(tsoErrCh) > 0 {
		tsoErr := <-tsoErrCh
		tsoErrors = append(tsoErrors, tsoErr)
	}
	errCh <- util2.WrapErrors(tsoErrors)
}

func (c *crossRegionClient) requestTSO(ctx context.Context, dcLocation string, wg *sync.WaitGroup, errCh chan<- error) {
	defer wg.Done()
	if c.pdClient != nil {
		for i := 0; i < c.TSORequestTimes; i++ {
			_, _, err := c.pdClient.GetLocalTS(ctx, dcLocation)
			if err != nil {
				errCh <- err
				return
			}
		}
	}
}

func buildPDSvcName(name, namespace string) string {
	return fmt.Sprintf("%s-pd.%s.svc:2379", name, namespace)
}

func buildAddr(addr string, port int32) string {
	return fmt.Sprintf("http://%s:%v", addr, port)
}
