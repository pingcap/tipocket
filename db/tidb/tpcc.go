package tidb

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/pingcap/go-tpc/pkg/workload"
	"github.com/pingcap/go-tpc/tpcc"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/history"
)

type tpccClient struct {
	*tpcc.Config
	db *sql.DB
	workload.Workloader
	workloadCtx context.Context
	asyncCommit bool
	onePC       bool
}

func (t *tpccClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	var (
		err  error
		node = clientNodes[idx]
	)
	t.db, err = sql.Open("mysql", fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port))
	if err != nil {
		return err
	}

	scopes := []string{"", "global."}
	varNames := []string{"tidb_enable_async_commit", "tidb_enable_1pc"}
	values := []int8{0, 0}
	if t.asyncCommit {
		values[0] = 1
	}
	if t.onePC {
		values[1] = 1
	}

	for id, varName := range varNames {
		for _, scope := range scopes {
			query := fmt.Sprintf("set @@%s%s = %v;", scope, varName, values[id])
			if _, err := t.db.Exec(query); err != nil {
				log.Fatalf("[tpcc] set %s failed: %v", varName, err)
			}
		}
	}

	t.Workloader = tpcc.NewWorkloader(t.db, t.Config)
	t.workloadCtx = t.Workloader.InitThread(ctx, 0)
	if idx == 0 {
		if err := t.Workloader.Cleanup(t.workloadCtx, 0); err != nil {
			log.Fatalf("cleanuping tpcc failed: %+v", err)
		}
		if err := t.Workloader.Prepare(t.workloadCtx, 0); err != nil {
			log.Fatalf("preparing tpcc failed: %+v", err)
		}
		if err := t.Workloader.CheckPrepare(t.workloadCtx, 0); err != nil {
			log.Fatalf("checking prepare failed: %+v", err)
		}
	}
	return nil
}

func (t *tpccClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	defer t.CleanupThread(t.workloadCtx, 0)
	if idx == 0 {
		return t.Workloader.Cleanup(t.workloadCtx, 0)
	}
	return nil
}

func (t *tpccClient) ScheduledClientExtensions() core.OnScheduleClientExtensions {
	return t
}

func (t *tpccClient) AutoDriveClientExtensions() core.AutoDriveClientExtensions {
	return nil
}

func (t *tpccClient) Invoke(ctx context.Context, node cluster.ClientNode, r interface{}) (response core.UnknownResponse) {
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
			worker := tpcc.NewWorkloader(t.db, t.Config)
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

func (t *tpccClient) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	return nil
}

// TPCCClientCreator creates tpccClient
type TPCCClientCreator struct {
	*tpcc.Config
	tpccClient  []*tpccClient
	AsyncCommit bool
	OnePC       bool
}

// Create ...
func (t *TPCCClientCreator) Create(_ cluster.ClientNode) core.Client {
	client := &tpccClient{
		Config:      t.Config,
		asyncCommit: t.AsyncCommit,
		onePC:       t.OnePC,
	}
	t.tpccClient = append(t.tpccClient, client)
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

// TPCCParser ...
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

// TPCCChecker ...
type TPCCChecker struct {
	CreatorRef *TPCCClientCreator
}

// Check ...
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

// Name ...
func (t *TPCCChecker) Name() string {
	return "tpcc"
}
