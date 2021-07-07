package staleread

import (
	"context"
	"fmt"
	"time"

	"github.com/pingcap/log"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/tiancaiamao/sysbench"
	"go.uber.org/zap"
)

// Config ...
type Config struct {
	SysBenchWorkerCount int
	SysBenchDuration    time.Duration
	RowsEachInsert      int
	InsertCount         int
	PreSec              int
}

// ClientCreator ...
type ClientCreator struct {
	Config
}

// Create ...
func (l ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &staleReadClient{
		workerCount:    l.SysBenchWorkerCount,
		duration:       l.SysBenchDuration,
		insertCount:    l.InsertCount,
		rowsEachInsert: l.RowsEachInsert,
		initialized:    false,
		preSecs:        l.PreSec,
	}
}

type staleReadClient struct {
	workerCount    int
	duration       time.Duration
	rowsEachInsert int
	insertCount    int
	sysBenchConf   *sysbench.Config
	promCli        api.Client
	initialized    bool
	preSecs        int
}

// SetUp...
func (c *staleReadClient) SetUp(ctx context.Context, _ []cluster.Node, cnodes []cluster.ClientNode, idx int) error {
	if c.initialized {
		return nil
	}
	name := cnodes[idx].ClusterName
	namespace := cnodes[idx].Namespace
	sysCase := &SysbenchCase{
		rowsEachInsert: c.rowsEachInsert,
		insertCount:    c.insertCount,
		preSec:         c.preSecs,
	}
	c.sysBenchConf = &sysbench.Config{
		Conn: sysbench.ConnConfig{
			User: "root",
			Host: buildTiDBHost(name, namespace),
			Port: 4000,
			DB:   "test",
		},
		Prepare: sysbench.PrepareConfig{
			WorkerCount: 1,
			Task:        sysCase,
		},
		Run: sysbench.RunConfig{
			WorkerCount: c.workerCount,
			Duration:    c.duration,
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
	log.Info("stale read testcase setup success")
	return nil
}

// Start ...
func (c *staleReadClient) Start(ctx context.Context, cfg interface{}, cnodes []cluster.ClientNode) error {
	log.Info("Run sysbench test")
	sysbench.RunTest(c.sysBenchConf)
	log.Info("Run sysbench test success")
	v1api := v1.NewAPI(c.promCli)
	value, _, err := v1api.QueryRange(ctx, `pd_scheduler_store_status{store=~".*", type="store_read_rate_keys"}`, v1.Range{Start: time.Now().Add(-30 * time.Second), End: time.Now(), Step: 15 * time.Second})
	if err != nil {
		log.Info("query prometheus failed", zap.Error(err))
		return err
	}
	smvms := make(map[string][]StoreMetricsValue, 0)
	switch v := value.(type) {
	case model.Matrix:
		for _, stream := range v {
			storeID := stream.Metric["store"]
			smv := make([]StoreMetricsValue, 0)
			for _, row := range stream.Values {
				smv = append(smv, StoreMetricsValue{
					Timestamp: row.Timestamp.Unix(),
					Value:     float64(row.Value),
					StoreID:   string(storeID),
				})
			}
			smvms[string(storeID)] = smv
		}
	default:
		log.Warn("mode value transform error")
	}
	// assert metrics here
	handleStoreMetricsValue(smvms, c.duration)
	return nil
}

// TearDown...
func (c *staleReadClient) TearDown(ctx context.Context, _ []cluster.ClientNode, idx int) error {
	return nil
}

func buildTiDBHost(name, namespace string) string {
	return fmt.Sprintf("%s-tidb.%s.svc", name, namespace)
}

func buildPrometheusSvcName(name, namespace string) string {
	return fmt.Sprintf("%s-prometheus.%s.svc:9090", name, namespace)
}

// StoreMetricsValue indicates the metrics value for a store from prometheus
type StoreMetricsValue struct {
	Timestamp int64
	Value     float64
	StoreID   string
}

// TODO: assert store read bytes/keys metrics
func handleStoreMetricsValue(smvms map[string][]StoreMetricsValue, duration time.Duration) {

}
