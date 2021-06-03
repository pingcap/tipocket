package staleread

import (
	"context"
	"fmt"
	"github.com/pingcap/log"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.uber.org/zap"
	"time"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/tiancaiamao/sysbench"
)

type ClientCreator struct {
}

// Create ...
func (l ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &staleReadClient{
		initialized: false,
	}
}

type staleReadClient struct {
	sysBenchConf *sysbench.Config
	promCli      api.Client
	initialized  bool
}

// SetUp...
func (c *staleReadClient) SetUp(ctx context.Context, _ []cluster.Node, cnodes []cluster.ClientNode, idx int) error {
	if c.initialized {
		return nil
	}
	name := cnodes[idx].ClusterName
	namespace := cnodes[idx].Namespace
	sysCase := &SysbenchCase{}
	sysCase.dbHost = buildTiDBSvcName(name, namespace)
	sysCase.rowsEachInsert = 50
	sysCase.insertCount = 50
	c.sysBenchConf = &sysbench.Config{
		Conn: sysbench.DefaultConnConfig(),
		Prepare: sysbench.PrepareConfig{
			WorkerCount: 1,
			Task:        sysCase,
		},
		Run: sysbench.RunConfig{
			WorkerCount: 4,
			Duration:    5 * time.Minute,
			Task:        sysCase,
		},
		Cleanup: sysCase,
	}

	cfg := api.Config{Address: fmt.Sprintf("http://%s", buildPrometheusSvcName(name, namespace))}
	promCli, err := api.NewClient(cfg)
	if err != nil {
		return err
	}
	c.promCli = promCli
	c.initialized = true
	return nil
}

// SetUp...
func (c *staleReadClient) Start(ctx context.Context, cfg interface{}, cnodes []cluster.ClientNode) error {
	sysbench.RunTest(c.sysBenchConf)
	v1api := v1.NewAPI(c.promCli)
	v, _, err := v1api.QueryRange(ctx, "", v1.Range{Start: time.Now().Add(-5 * time.Minute), End: time.Now(), Step: 15 * time.Second})
	if err != nil {
		return err
	}
	log.Info("metrics", zap.String("metrics", v.String()))
	return nil
}

// TearDown...
func (c *staleReadClient) TearDown(ctx context.Context, _ []cluster.ClientNode, idx int) error {
	return nil
}

func buildTiDBSvcName(name, namespace string) string {
	return fmt.Sprintf("%s-tidb.%s.svc:4000", name, namespace)
}

func buildPrometheusSvcName(name, namespace string) string {
	return fmt.Sprintf("%s-prometheus.%s.svc:9090", name, namespace)
}
