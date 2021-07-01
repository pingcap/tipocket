package rwregister

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	ellecore "github.com/pingcap/tipocket/pkg/elle/core"
	elleregister "github.com/pingcap/tipocket/pkg/elle/rw_register"
	elletxn "github.com/pingcap/tipocket/pkg/elle/txn"
)

type registerResponse struct {
	Result ellecore.Op `json:result`
}

func (c registerResponse) String() string {
	return c.Result.String()
}

// IsUnknown always returns false because we don't want to let it be resorted
func (c registerResponse) IsUnknown() bool {
	return false
}

type client struct {
	tableCount  int
	useIndex    bool
	readLock    string
	txnMode     string
	replicaRead string

	db          *sql.DB
	nextRequest func() ellecore.Op
}

func (c *client) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	var err error
	txnMode := c.txnMode
	if txnMode == "mixed" {
		switch rand.Intn(2) {
		case 0:
			txnMode = "pessimistic"
		default:
			txnMode = "optimistic"
		}
	}
	if txnMode != "optimistic" && txnMode != "pessimistic" {
		return fmt.Errorf("illegal txn_mode value: %s", txnMode)
	}

	node := clientNodes[idx]
	c.db, err = sql.Open("mysql", fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port))
	if err != nil {
		return err
	}
	if _, err := c.db.Exec(fmt.Sprintf("set @@tidb_txn_mode = '%s'", txnMode)); err != nil {
		return fmt.Errorf("set tidb_txn_mode failed: %v", err)
	}
	if c.replicaRead != "" {
		if _, err := c.db.Exec(fmt.Sprintf("set @@tidb_replica_read = '%s'", c.replicaRead)); err != nil {
			return fmt.Errorf("set tidb_replica_read failed: %v", err)
		}
	}
	time.Sleep(3 * time.Second)
	return nil
}

func (c *client) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}
	sql := `drop table if exists register`
	if _, err := c.db.Exec(sql); err != nil {
		return err
	}
	return c.db.Close()
}

func (c *client) Invoke(ctx context.Context, node cluster.ClientNode, r interface{}) core.UnknownResponse {
	request := r.(ellecore.Op)
	txn, err := c.db.Begin()
	if err != nil {
		return registerResponse{
			Result: ellecore.Op{
				Time:  time.Now(),
				Type:  ellecore.OpTypeFail,
				Value: request.Value,
				Error: err.Error(),
			},
		}
	}
	var mops []ellecore.Mop
	for _, mop := range *request.Value {
		switch mop.GetMopType() {
		case ellecore.MopTypeWrite:
			k := mop.GetKey()
			v := mop.GetValue().(elleregister.Int).MustGetVal()
			_, err := txn.ExecContext(ctx, "insert into register(id, sk, val) values (?, ?, ?) on duplicate key update val = ?", k, k, v, v)
			if err != nil {
				_ = txn.Rollback()
				return registerResponse{
					Result: ellecore.Op{
						Time:  time.Now(),
						Type:  ellecore.OpTypeFail,
						Value: request.Value,
						Error: err.Error(),
					},
				}
			}
			mops = append(mops, ellecore.Mop{
				T: ellecore.MopTypeWrite,
				M: map[string]interface{}{
					"key":   k,
					"value": elleregister.NewInt(v),
				},
			})
		case ellecore.MopTypeRead:
			k := mop.GetKey()
			column := "id"
			if c.useIndex {
				column = "sk"
			}
			query := fmt.Sprintf("select val from register where %s = ?", column)
			if c.readLock != "" {
				query += " " + c.readLock
			}
			rows, err := txn.QueryContext(ctx, query, k)
			if err != nil {
				_ = txn.Rollback()
				return registerResponse{
					Result: ellecore.Op{
						Time:  time.Now(),
						Type:  ellecore.OpTypeFail,
						Value: request.Value,
						Error: err.Error(),
					},
				}
			}
			var value string
			if rows.Next() {
				if err := rows.Scan(&value); err != nil {
					rows.Close()
					return registerResponse{
						Result: ellecore.Op{
							Time:  time.Now(),
							Type:  ellecore.OpTypeFail,
							Value: request.Value,
							Error: err.Error(),
						},
					}
				}
			}
			rows.Close()
			var v elleregister.Int
			if value == "" {
				v = elleregister.NewNil()
			} else {
				v = elleregister.NewInt(mustAtoi(value))
			}
			mops = append(mops, ellecore.Mop{
				T: ellecore.MopTypeRead,
				M: map[string]interface{}{
					"key":   k,
					"value": v,
				},
			})
		}
	}

	if err := txn.Commit(); err != nil {
		tp := ellecore.OpTypeFail
		if strings.Contains(err.Error(), "invalid connection") {
			tp = ellecore.OpTypeUnknown
		}
		return registerResponse{
			Result: ellecore.Op{
				Time:  time.Now(),
				Type:  tp,
				Value: &mops,
				Error: err.Error(),
			},
		}
	}

	return registerResponse{
		Result: ellecore.Op{
			Time:  time.Now(),
			Type:  ellecore.OpTypeOk,
			Value: &mops,
		},
	}
}

func (c *client) NextRequest() interface{} {
	return c.nextRequest()
}

func (c *client) DumpState(_ context.Context) (interface{}, error) {
	if _, err := c.db.Exec("drop table if exists register"); err != nil {
		return nil, err
	}
	if _, err := c.db.Exec(`create table if not exists register
		(id  int not null primary key,
		 sk  int not null,
		 val int not null)`); err != nil {
		return nil, err
	}
	if c.useIndex {
		if _, err := c.db.Exec(`create index txn_sk_val on register(sk, val)`); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// ClientCreator can create list append client
type registerClientCreator struct {
	tableCount  int
	readLock    string
	txnMode     string
	replicaRead string

	it *elletxn.MopIterator
	mu sync.Mutex
}

// NewClientCreator ...
func NewClientCreator(tableCount int, readLock, txnMode, replicaRead string) core.ClientCreator {
	opt := elletxn.DefaultWrTxnOpts()
	opt.MaxWritesPerKey = 1024
	return &registerClientCreator{
		tableCount:  tableCount,
		readLock:    readLock,
		txnMode:     txnMode,
		replicaRead: replicaRead,
		it:          elletxn.WrTxn(opt),
	}
}

// Create creates a client.
func (r *registerClientCreator) Create(_ cluster.ClientNode) core.Client {
	return &client{
		tableCount: r.tableCount,
		readLock:   r.readLock,
		txnMode:    r.txnMode,
		nextRequest: func() ellecore.Op {
			r.mu.Lock()
			defer r.mu.Unlock()
			value := r.it.Next()
			for index, mop := range value {
				var intVal elleregister.Int
				switch v := mop.M["value"].(type) {
				case int:
					intVal = elleregister.NewInt(v)
				case nil:
					intVal = elleregister.NewNil()
				default:
					panic("unexpected type")
				}
				mop.M["value"] = intVal
				if mop.T == ellecore.MopTypeAppend {
					value[index].T = ellecore.MopTypeWrite
				}
				value[index].M = mop.M
			}
			return ellecore.Op{
				Type:  ellecore.OpTypeInvoke,
				Time:  time.Now(),
				Value: &value,
			}
		},
	}
}

// RegisterParser ...
type RegisterParser struct{}

// OnRequest ...
func (RegisterParser) OnRequest(data json.RawMessage) (interface{}, error) {
	r := ellecore.Op{}
	str := string(data)
	_ = str
	err := json.Unmarshal(data, &r)
	return r, err
}

// OnResponse ...
func (a RegisterParser) OnResponse(data json.RawMessage) (interface{}, error) {
	r := registerResponse{}
	err := json.Unmarshal(data, &r)
	return r, err
}

// OnNoopResponse ...
func (a RegisterParser) OnNoopResponse() interface{} {
	panic("unreachable")
}

// OnState ...
func (a RegisterParser) OnState(state json.RawMessage) (interface{}, error) {
	return nil, nil
}

// RegisterChecker ...
type RegisterChecker struct{}

// Check ...
func (RegisterChecker) Check(_ core.Model, ops []core.Operation) (bool, error) {
	history := convertOperationsToHistory(ops)
	_ = writeEdnHistory(history)

	result := elleregister.Check(
		elletxn.Opts{Anomalies: []string{"G-single"}},
		history,
		elleregister.GraphOption{},
	)
	if result.Valid {
		return true, nil
	}
	return false, result
}

// Name ...
func (a RegisterChecker) Name() string {
	return "rw_register"
}

func mustAtoi(s string) int {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		log.Panicf("parse int failed: %s", s)
	}
	return int(i)
}

func convertOperationsToHistory(events []core.Operation) ellecore.History {
	var history ellecore.History
	for _, event := range events {
		var op ellecore.Op
		switch e := event.Data.(type) {
		case ellecore.Op:
			op = e
			op.Process.Set(int(event.Proc))
		case registerResponse:
			op = e.Result
			op.Process.Set(int(event.Proc))
		default:
			panic("unreachable")
		}
		mops := op.Value
		typedMops := make([]ellecore.Mop, 0)
		for _, mop := range *mops {
			var v elleregister.Int
			val := mop.GetValue().(map[string]interface{})
			if val["is_num"].(bool) {
				v = elleregister.NewNil()
			} else {
				v = elleregister.NewInt(int(val["val"].(float64)))
			}
			m := ellecore.Mop{
				M: map[string]interface{}{
					"key":   mop.GetKey(),
					"value": v,
				},
			}
			if mop.IsRead() {
				m.T = ellecore.MopTypeRead
				typedMops = append(typedMops, m)
			}
			if mop.IsWrite() {
				m.T = ellecore.MopTypeWrite
				typedMops = append(typedMops, m)
			}
		}
		op.Value = &typedMops
		history = append(history, op)
	}
	return history
}

func writeEdnHistory(history ellecore.History) error {
	history.AttachIndexIfNoExists()
	f, err := os.Create("history.edn")
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	for _, op := range history {
		w.WriteString(op.String())
		w.WriteString("\n")
	}
	return w.Flush()
}
