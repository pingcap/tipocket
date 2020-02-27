package tidb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"

	"github.com/pingcap/go-tpc/pkg/workload"
	"github.com/pingcap/go-tpc/tpcc"

	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"
)

var tpccConfig = tpcc.Config{
	Threads:    1,
	Parts:      1,
	Warehouses: 1,
	UseFK:      false,
	Isolation:  0,
	CheckAll:   false,
}

type tpccClient struct {
	db *sql.DB
	workload.Workloader
	workloadCtx context.Context
}

func (t *tpccClient) SetUp(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error {
	var (
		err  error
		node = nodes[idx]
	)
	t.db, err = sql.Open("mysql", fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port))
	if err != nil {
		return err
	}
	t.Workloader = tpcc.NewWorkloader(t.db, &tpccConfig)
	t.workloadCtx = t.Workloader.InitThread(ctx, 0)
	if idx == 0 {
		if err := t.Workloader.Prepare(t.workloadCtx, 0); err != nil {
			log.Fatalf("preparing tpcc failed: %+v", err)
		}
		if err := t.Workloader.CheckPrepare(t.workloadCtx, 0); err != nil {
			log.Fatalf("checking prepare failed: %+v", err)
		}
	}
	return nil
}

func (t *tpccClient) TearDown(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error {
	defer t.CleanupThread(t.workloadCtx, 0)
	if idx == 0 {
		return t.Workloader.Cleanup(t.workloadCtx, 0)
	}
	return nil
}

func (t *tpccClient) Invoke(ctx context.Context, node clusterTypes.ClientNode, r interface{}) (response interface{}) {
	s := time.Now()
	err := t.Workloader.Run(t.workloadCtx, 0)
	// TPCC.InitThread will panic when establish connection failed
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("recovered from ", r)
			response = tpccResponse{Duration: time.Now().Sub(s), Error: err.Error()}
		}
	}()
	if err != nil {
		if strings.Contains(err.Error(), "connection is already closed") {
			// reconnect
			log.Printf("connecting...")
			worker := tpcc.NewWorkloader(t.db, &tpccConfig)
			nCtx := t.Workloader.InitThread(context.TODO(), 0)
			log.Printf("reconnected")
			t.Workloader = worker
			t.workloadCtx = nCtx
		}
		return tpccResponse{Duration: time.Now().Sub(s), Error: err.Error()}
	}
	return tpccResponse{Duration: time.Now().Sub(s)}
}

func (t *tpccClient) Check() error {
	return t.Workloader.Check(t.workloadCtx, 0)
}

func (t *tpccClient) NextRequest() interface{} {
	return tpccRequest{}
}

func (t tpccClient) DumpState(ctx context.Context) (interface{}, error) {
	return nil, nil
}

func (t *tpccClient) Start(ctx context.Context, cfg interface{}, clientNodes []clusterTypes.ClientNode) error {
	return nil
}

// TPCCClientCreator creates tpccClient
type TPCCClientCreator struct {
	tpccClient []*tpccClient
}

func (T *TPCCClientCreator) Create(node clusterTypes.ClientNode) core.Client {
	client := &tpccClient{}
	T.tpccClient = append(T.tpccClient, client)
	return client
}

type tpccRequest struct{}

// tpccResponse implements core.UnknownResponse
type tpccResponse struct {
	time.Duration
	Error string `json:",omitempty"`
}

func (t tpccResponse) IsUnknown() bool {
	// we don't care isUnknown here
	return false
}

// tpccParser implements history.RecordParser
type tpccParser struct {
}

func TPCCParser() history.RecordParser {
	return tpccParser{}
}

func (t tpccParser) OnRequest(data json.RawMessage) (interface{}, error) {
	r := tpccRequest{}
	err := json.Unmarshal(data, &r)
	return r, err
}

func (t tpccParser) OnResponse(data json.RawMessage) (interface{}, error) {
	r := tpccResponse{}
	err := json.Unmarshal(data, &r)
	return r, err
}

func (t tpccParser) OnNoopResponse() interface{} {
	panic("unreachable")
}

func (t tpccParser) OnState(state json.RawMessage) (interface{}, error) {
	return nil, nil
}

type TPCCChecker struct {
	CreatorRef *TPCCClientCreator
}

func (t *TPCCChecker) Check(m core.Model, ops []core.Operation) (bool, error) {
	for _, client := range t.CreatorRef.tpccClient {
		if err := client.Check(); err != nil {
			// ignore 2.10 and 2.12 consistency conditions
			if !strings.Contains(err.Error(), "3.3.2.10") &&
				!strings.Contains(err.Error(), "3.3.2.12") {
				return false, err
			}
		}
		return true, nil
	}
	return false, nil
}

func (t *TPCCChecker) Name() string {
	return "tpcc"
}
