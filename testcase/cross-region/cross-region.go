package crossregion

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/util/pdutil"
	"github.com/pingcap/tipocket/util"
	pdClient "github.com/tikv/pd/client"
)

type Config struct {
	DBName string
}

// ClientCreator creates ondupClient
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
	return nil
}

func (c *crossRegionClient) TearDown(ctx context.Context, _ []cluster.ClientNode, idx int) error {
	if c.db != nil {
		c.db.Close()
	}
	if c.pdClient != nil {
		c.pdClient.Close()
	}
	//c.pdClient.Close()
	return nil
}

func (c *crossRegionClient) Start(ctx context.Context, cfg interface{}, cnodes []cluster.ClientNode) error {
	_, err := c.pdHttpClient.GetStores()
	if err != nil {
		return err
	}
	_, err = c.db.Begin()
	if err != nil {
		return err
	}
	return nil
}

func buildPDSvcName(name, namespace string) string {
	return fmt.Sprintf("%s-pd.%s.svc:2379", name, namespace)
}

func buildAddr(addr string, port int32) string {
	return fmt.Sprintf("http://%s:%v", addr, port)
}
