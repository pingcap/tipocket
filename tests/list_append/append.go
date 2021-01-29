package listappend

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
	elleappend "github.com/pingcap/tipocket/pkg/elle/list_append"
	elletxn "github.com/pingcap/tipocket/pkg/elle/txn"
)

type appendResponse struct {
	Result ellecore.Op `json:result`
}

func (c appendResponse) String() string {
	return c.Result.String()
}

// IsUnknown always returns false because we don't want to let it be resorted
func (c appendResponse) IsUnknown() bool {
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
	for i := 0; i < c.tableCount; i++ {
		sql := fmt.Sprintf(`drop table if exists txn_%d`, i)
		if _, err := c.db.Exec(sql); err != nil {
			return err
		}
	}

	return c.db.Close()
}

func (c *client) Invoke(ctx context.Context, node cluster.ClientNode, r interface{}) core.UnknownResponse {
	request := r.(ellecore.Op)
	txn, err := c.db.Begin()
	if err != nil {
		return appendResponse{
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
		case ellecore.MopTypeAppend:
			k := mop.GetKey()
			v := mop.GetValue().(int)
			table := mustAtoi(k) % c.tableCount
			// need set a timeout here?
			_, err := txn.ExecContext(ctx, fmt.Sprintf("insert into txn_%d(id, sk, val) values (?, ?, ?) on duplicate key update val = CONCAT(val, ',', ?)",
				table), k, k, v, v)
			if err != nil {
				_ = txn.Rollback()
				return appendResponse{
					Result: ellecore.Op{
						Time:  time.Now(),
						Type:  ellecore.OpTypeFail,
						Value: request.Value,
						Error: err.Error(),
					},
				}
			}
			mops = append(mops, ellecore.Append(
				k,
				v,
			))
		case ellecore.MopTypeRead:
			k := mop.GetKey()
			table := mustAtoi(k) % c.tableCount
			column := "id"
			if c.useIndex {
				column = "sk"
			}
			query := fmt.Sprintf("select val from txn_%d where %s = ?", table, column)
			if c.readLock != "" {
				query += " " + c.readLock
			}
			rows, err := txn.QueryContext(ctx, query, k)
			if err != nil {
				_ = txn.Rollback()
				return appendResponse{
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
					return appendResponse{
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
			var v []int
			if len(value) != 0 {
				for _, n := range strings.Split(value, ",") {
					v = append(v, mustAtoi(n))
				}
			}
			mops = append(mops, ellecore.Read(
				k,
				v,
			))
		}
	}

	if err := txn.Commit(); err != nil {
		tp := ellecore.OpTypeFail
		if strings.Contains(err.Error(), "invalid connection") {
			tp = ellecore.OpTypeUnknown
		}
		return appendResponse{
			Result: ellecore.Op{
				Time:  time.Now(),
				Type:  tp,
				Value: &mops,
				Error: err.Error(),
			},
		}
	}

	return appendResponse{
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
	for i := 0; i < c.tableCount; i++ {
		if _, err := c.db.Exec(fmt.Sprintf("drop table if exists txn_%d", i)); err != nil {
			return nil, err
		}
		if _, err := c.db.Exec(fmt.Sprintf(`create table if not exists txn_%d
			(id     int not null primary key,
			sk int not null,
			val text)`, i)); err != nil {
			return nil, err
		}
		if c.useIndex {
			if _, err := c.db.Exec(fmt.Sprintf(`create index txn_%d_sk_val on txn_%d (sk, val)`, i, i)); err != nil {
				return nil, err
			}
		}
	}
	return nil, nil
}

// ClientCreator can create list append client
type appendClientCreator struct {
	tableCount  int
	readLock    string
	txnMode     string
	replicaRead string

	it *elletxn.MopIterator
	mu sync.Mutex
}

// NewClientCreator ...
func NewClientCreator(tableCount int, readLock, txnMode, replicaRead string) core.ClientCreator {
	return &appendClientCreator{
		tableCount:  tableCount,
		readLock:    readLock,
		txnMode:     txnMode,
		replicaRead: replicaRead,
		it:          elletxn.WrTxnWithDefaultOpts(),
	}
}

// Create creates a client.
func (a *appendClientCreator) Create(_ cluster.ClientNode) core.Client {
	return &client{
		tableCount:  a.tableCount,
		readLock:    a.readLock,
		txnMode:     a.txnMode,
		replicaRead: a.replicaRead,
		nextRequest: func() ellecore.Op {
			a.mu.Lock()
			defer a.mu.Unlock()
			value := a.it.Next()
			return ellecore.Op{
				Type:  ellecore.OpTypeInvoke,
				Time:  time.Now(),
				Value: &value,
			}
		},
	}
}

// AppendParser ...
type AppendParser struct{}

// OnRequest ...
func (a AppendParser) OnRequest(data json.RawMessage) (interface{}, error) {
	r := ellecore.Op{}
	str := string(data)
	_ = str
	err := json.Unmarshal(data, &r)
	return r, err
}

// OnResponse ...
func (a AppendParser) OnResponse(data json.RawMessage) (interface{}, error) {
	r := appendResponse{}
	err := json.Unmarshal(data, &r)
	return r, err
}

// OnNoopResponse ...
func (a AppendParser) OnNoopResponse() interface{} {
	panic("unreachable")
}

// OnState ...
func (a AppendParser) OnState(state json.RawMessage) (interface{}, error) {
	return nil, nil
}

// AppendChecker ...
type AppendChecker struct{}

// Check ...
func (a AppendChecker) Check(_ core.Model, ops []core.Operation) (bool, error) {
	history := convertOperationsToHistory(ops)
	_ = writeEdnHistory(history)

	result := elleappend.Check(
		elletxn.Opts{Anomalies: []string{"G-single"}},
		history)
	if result.Valid {
		return true, nil
	}
	return false, result
}

// Name ...
func (a AppendChecker) Name() string {
	return "list_append"
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
		case appendResponse:
			op = e.Result
			op.Process.Set(int(event.Proc))
		default:
			panic("unreachable")
		}
		mops := op.Value
		typedMops := make([]ellecore.Mop, 0)
		for _, mop := range *mops {
			if mop.IsRead() {
				var value []int
				if mop.GetValue() != nil {
					for _, v := range mop.GetValue().([]interface{}) {
						value = append(value, int(v.(float64)))
					}
					typedMops = append(typedMops, ellecore.Read(mop.GetKey(), value))
				} else {
					typedMops = append(typedMops, ellecore.Read(mop.GetKey(), nil))
				}
			}
			if mop.IsAppend() {
				typedMops = append(typedMops, ellecore.Append(mop.GetKey(), int(mop.GetValue().(float64))))
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
