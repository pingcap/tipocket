package pipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql" // mysql

	"github.com/ngaut/log"
	"github.com/pingcap/errors"
	"github.com/pingcap/tidb/tablecodec"
	"github.com/pingcap/tidb/util/codec"
	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/nemesis"
	httputil "github.com/pingcap/tipocket/pkg/util/http"
	"github.com/pingcap/tipocket/util"
)

const (
	dbName    = "pipeline"
	tableName = "t"
)

type Config struct {
	TableSize        int
	ContenderCount   int
	ScheduleInterval time.Duration
	ReportInterval   time.Duration
	Strict           bool
	LocalMode        bool
}

func (c *Config) Normalize() *Config {
	if c.TableSize == 0 {
		c.TableSize = 1
	}
	if c.ContenderCount == 0 {
		c.ContenderCount = 10
	}
	if c.ScheduleInterval <= time.Second {
		c.ScheduleInterval = time.Second
	}
	return c
}

func (c *Config) Create(node cluster.ClientNode) core.Client {
	return &pipelineClient{
		Config: c.Normalize(),
	}
}

type pipelineClient struct {
	*Config

	dbAddr       string
	dbStatusAddr string
	pdAddr       string

	db        *sql.DB
	scheduler *nemesis.LeaderShuffler

	// stats
	lastTotal   uint64
	lastErrors  uint64
	total       uint64
	errors      uint64
	reportCount uint64
}

func (c *pipelineClient) openDB(ctx context.Context) error {
	dsn := fmt.Sprintf("root@tcp(%s)/", c.dbAddr)
	db, err := util.OpenDB(dsn, 1)
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()
	util.MustExec(db, `CREATE DATABASE IF NOT EXISTS `+dbName)
	c.db, err = util.OpenDB(dsn+"pipeline", 100)
	return errors.Trace(err)
}

func (c *pipelineClient) setUpAddrs(dbNode cluster.ClientNode, pdNode cluster.Node) {
	if c.LocalMode {
		c.dbAddr = "127.0.0.1:4000"
		c.dbStatusAddr = "127.0.0.1:10080"
		c.pdAddr = "127.0.0.1:2379"
	} else {
		c.dbAddr = dbNode.Address()
		c.dbStatusAddr = fmt.Sprintf("%s-tidb.%s.svc:10080", dbNode.ClusterName, dbNode.Namespace)
		c.pdAddr = fmt.Sprintf("%s-pd.%s.svc:2379", pdNode.ClusterName, pdNode.Namespace)
	}
}

// SetUp sets up the client.
func (c *pipelineClient) SetUp(ctx context.Context, nodes []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	c.setUpAddrs(clientNodes[0], nodes[0])

	if err := c.openDB(ctx); err != nil {
		return errors.Trace(err)

	}
	util.MustExec(c.db, `set @@global.tidb_txn_mode='pessimistic'`)
	util.MustExec(c.db, `DROP TABLE IF EXISTS `+tableName)
	util.MustExec(c.db, fmt.Sprintf(`CREATE TABLE %s (id INT PRIMARY KEY, v BIGINT)`, tableName))
	for i := 0; i < c.TableSize; i++ {
		util.MustExec(c.db, fmt.Sprintf(`insert into %s values (%d, 0)`, tableName, i))
	}

	startKey, _, err := c.getTableRange(dbName, tableName)
	if err != nil {
		return errors.Trace(err)
	}
	c.scheduler = nemesis.NewLeaderShuffler(string(startKey))
	c.scheduler.SetPDAddr("http://" + c.pdAddr)
	return nil
}

func (c *pipelineClient) getTableRange(db, table string) (startKey, endKey []byte, err error) {
	url := fmt.Sprintf("http://%s/schema/%s/%s", c.dbStatusAddr, db, table)
	resp, err := httputil.NewHTTPClient(http.DefaultClient).Get(url)
	if err != nil {
		return
	}
	var body struct {
		ID int64 `json:"id"`
	}
	err = json.Unmarshal(resp, &body)
	if err != nil {
		return
	}
	startKey, endKey = tablecodec.GetTableHandleKeyRange(body.ID)
	startKey = codec.EncodeBytes([]byte{}, startKey)
	endKey = codec.EncodeBytes([]byte{}, endKey)
	return
}

// TearDown tears down the client.
func (c *pipelineClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	c.db.ExecContext(ctx, `DROP DATABASE IF EXISTS `+dbName)
	return c.db.Close()
}

// Invoke invokes a request to the database.
// Mostly, the return Response should implement UnknownResponse interface
func (c *pipelineClient) Invoke(ctx context.Context, node cluster.ClientNode, r interface{}) core.UnknownResponse {
	panic("not implemented")
}

// NextRequest generates a request for latter Invoke.
func (c *pipelineClient) NextRequest() interface{} {
	panic("not implemented")
}

// DumpState the database state(also the model's state)
func (c *pipelineClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("not implemented")
}

// Start runs self scheduled cases
// this function will block Invoke trigger
// if you want to schedule cases by yourself, use this function only
func (c *pipelineClient) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	c.scatterTable(dbName, tableName)
	defer func() {
		c.reportSummary()
		c.stopScatterTable(dbName, tableName)
	}()

	for i := 0; i < c.TableSize; i++ {
		for j := 0; j < c.ContenderCount; j++ {
			go c.inc(ctx, i)
		}
	}

	scheduleTick := time.NewTicker(c.ScheduleInterval)
	reportTick := time.NewTicker(c.ReportInterval)
	for {
		select {
		case <-ctx.Done():
			return nil

		case <-scheduleTick.C:
			c.schedule(ctx)

		case <-reportTick.C:
			c.reportStats()
		}
	}
}

func (c *pipelineClient) scatterTable(db, table string) error {
	url := fmt.Sprintf("http://%s/tables/%s/%s/scatter", c.dbStatusAddr, db, table)
	_, err := httputil.NewHTTPClient(http.DefaultClient).Get(url)
	return err
}

func (c *pipelineClient) stopScatterTable(db, table string) error {
	url := fmt.Sprintf("http://%s/tables/%s/%s/stop-scatter", c.dbStatusAddr, db, table)
	_, err := httputil.NewHTTPClient(http.DefaultClient).Get(url)
	return err
}

func (c *pipelineClient) checkKeyIsLocked(ctx context.Context, row int) (bool, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	_, err = tx.QueryContext(ctx, fmt.Sprintf(`select * from %s where id = %d for update nowait`, tableName, row))
	if err == nil {
		return false, nil
	}
	if me, ok := err.(*mysql.MySQLError); !ok || me.Number != 3572 {
		return false, err
	}
	return true, nil
}

func (c *pipelineClient) inc(ctx context.Context, row int) {
	var cnt uint64
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		tx, err := c.db.Begin()
		if err != nil {
			log.Fatalf("fail to begin a transaction: %v", err)
		}
		// When pipelined pessimistic locking is enabled, returning successful result means the lock request has been proposed.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET v = v + 1 WHERE id = %d`, tableName, row)); err != nil {
			log.Fatalf("fail to update: %v", err)
		}
		// Check whether the lock above is applied successfully.
		if locked, err := c.checkKeyIsLocked(ctx, row); !locked || err != nil {
			if c.Strict {
				log.Fatalf("fail to lock: %v", err)
			} else {
				log.Warnf("fail to lock: %v", err)
			}
		}
		if err := tx.Commit(); err != nil {
			if c.Strict {
				log.Fatalf("fail to commit: %v", err)
			} else {
				log.Warnf("fail to commit: %v", err)
			}
			atomic.AddUint64(&c.errors, 1)
		}
		cnt++
		if cnt == 10 {
			atomic.AddUint64(&c.total, cnt)
			cnt = 0
		}
	}
}

func (c *pipelineClient) schedule(ctx context.Context) {
	if err := c.scheduler.ShuffleLeader(); err != nil {
		log.Warnf("fail to schedule: %v", err)
	}
}

func (c *pipelineClient) reportStats() {
	c.reportCount++
	total := atomic.LoadUint64(&c.total)
	totalDiff := total - c.lastTotal
	c.lastTotal = total
	errors := atomic.LoadUint64(&c.errors)
	errorsDiff := errors - c.lastErrors
	c.lastErrors = errors

	log.Infof("[%v] tps: %.2f err/s: %.2f rate: %.2f%% errors/total: %d/%d(%.2f%%)",
		time.Duration(c.reportCount*uint64(c.ReportInterval)).String(),
		float64(totalDiff)/c.ReportInterval.Seconds(),
		float64(errorsDiff)/c.ReportInterval.Seconds(),
		float64(errorsDiff)/float64(totalDiff)*100,
		errors, total, float64(errors)/float64(total)*100,
	)
}

func (c *pipelineClient) reportSummary() {
	log.Infof("Summary: errors/total: %d/%d(%.2f%%)", atomic.LoadUint64(&c.errors), atomic.LoadUint64(&c.total), float64(c.errors)/float64(c.total)*100)
}
