package pipeline

import (
	"bytes"
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

// Config for this test.
type Config struct {
	TableSize        int
	ContenderCount   int
	ScheduleInterval time.Duration
	ReportInterval   time.Duration
	Strict           bool
	LocalMode        bool
}

func (c *Config) normalize() *Config {
	if c.TableSize == 0 {
		c.TableSize = 1
	}
	if c.ContenderCount == 0 {
		c.ContenderCount = 1
	}
	if c.ScheduleInterval <= time.Second {
		c.ScheduleInterval = time.Second
	}
	return c
}

// Create implements core.Client interface that creates a client for the test.
func (c *Config) Create(node cluster.ClientNode) core.Client {
	return &pipelineClient{
		Config: c.normalize(),
	}
}

type regionScheduler struct {
	*nemesis.LeaderShuffler
	// failpoints related to region scheduling.
	failpoints []string
}

func newRegionScheduler(pdAddr string, key string) *regionScheduler {
	leaderShuffler := nemesis.NewLeaderShuffler(pdAddr, key)
	return &regionScheduler{
		leaderShuffler,
		[]string{"apply_before_split", "apply_before_prepare_merge", "apply_before_commit_merge", "apply_before_rollback_merge"},
	}
}

func (s *regionScheduler) delayScheduling(kvStatusAddrs []string, delayMS int) error {
	client := httputil.NewHTTPClient(http.DefaultClient)
	// Add a delay to each failpoint so that more requests can fail due to region scheduling.
	for _, kvStatusAddr := range kvStatusAddrs {
		for _, fp := range s.failpoints {
			url := fmt.Sprintf("http://%s/fail/%s", kvStatusAddr, fp)
			_, err := client.Put(url, "", bytes.NewBufferString(fmt.Sprintf("delay(%d)", delayMS)))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *regionScheduler) schedule() {
	if err := s.ShuffleLeader(); err != nil {
		log.Warnf("fail to schedule: %v", err)
	}
}

type pipelineClient struct {
	*Config

	dbAddr        string
	dbStatusAddr  string
	pdAddr        string
	kvStatusAddrs []string

	db        *sql.DB
	scheduler *regionScheduler

	reportCount uint64
	// stats
	lastTotal     uint64
	lastLockErr   uint64
	lastCommitErr uint64
	total         uint64
	lockErr       uint64
	commitErr     uint64
}

func (c *pipelineClient) openDB(ctx context.Context) error {
	dsn := fmt.Sprintf("root@tcp(%s)/", c.dbAddr)
	db, err := util.OpenDB(dsn, 1)
	if err != nil {
		return errors.Trace(err)
	}
	defer db.Close()
	util.MustExec(db, `CREATE DATABASE IF NOT EXISTS `+dbName)
	c.db, err = util.OpenDB(dsn+"pipeline", c.TableSize*c.ContenderCount)
	return errors.Trace(err)
}

func (c *pipelineClient) setUpAddrs(dbNode cluster.ClientNode, nodes []cluster.Node) {
	var pdNode cluster.Node
	var kvNodes []cluster.Node

	for _, node := range nodes {
		switch node.Component {
		case cluster.PD:
			pdNode = node
		case cluster.TiKV:
			kvNodes = append(kvNodes, node)
		}
	}

	if c.LocalMode {
		c.dbAddr = "127.0.0.1:4000"
		c.dbStatusAddr = "127.0.0.1:10080"
		c.pdAddr = "127.0.0.1:2379"
		c.kvStatusAddrs = []string{"127.0.0.1:20180", "127.0.0.1:20181", "127.0.0.1:20182", "127.0.0.1:20183"}
	} else {
		c.dbAddr = dbNode.Address()
		c.dbStatusAddr = fmt.Sprintf("%s-tidb.%s.svc:10080", dbNode.ClusterName, dbNode.Namespace)
		c.pdAddr = fmt.Sprintf("%s-pd.%s.svc:2379", pdNode.ClusterName, pdNode.Namespace)
		for _, kvNode := range kvNodes {
			// ${POD_NAME}.${PEER_SERVICE_NAME}.${NAMESPACE}.svc
			c.kvStatusAddrs = append(c.kvStatusAddrs, fmt.Sprintf("%s.%s-tikv-peer.%s.svc:20180", kvNode.PodName, kvNode.ClusterName, kvNode.Namespace))
		}
	}
}

// SetUp sets up the client.
func (c *pipelineClient) SetUp(ctx context.Context, nodes []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	c.setUpAddrs(clientNodes[0], nodes)

	if err := c.openDB(ctx); err != nil {
		return errors.Trace(err)

	}
	util.MustExec(c.db, `set @@global.tidb_txn_mode='pessimistic'`)
	util.MustExec(c.db, `DROP TABLE IF EXISTS `+tableName)
	util.MustExec(c.db, fmt.Sprintf(`CREATE TABLE %s (id INT PRIMARY KEY, v BIGINT)`, tableName))
	// Each transaction modifies two rows.
	for i := 0; i < c.TableSize*2; i++ {
		util.MustExec(c.db, fmt.Sprintf(`insert into %s values (%d, 0)`, tableName, i))
	}

	startKey, _, err := c.getTableRange(dbName, tableName)
	if err != nil {
		return errors.Trace(err)
	}
	c.scheduler = newRegionScheduler("http://"+c.pdAddr, string(startKey))
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
	if err := c.scheduler.delayScheduling(c.kvStatusAddrs, 200); err != nil {
		log.Fatalf("failed to enable failpoints: %v", err)
	}
	if err := c.scatterTable(dbName, tableName); err != nil {
		log.Fatalf("failed to scatter table: %v", err)
	}

	defer func() {
		c.checkSum(context.Background())
		c.reportSummary()
		c.stopScatterTable(dbName, tableName)
	}()

	for i := 0; i < c.TableSize; i++ {
		for j := 0; j < c.ContenderCount; j++ {
			go c.transfer(ctx, i)
		}
	}

	checkTick := time.NewTicker(time.Second)
	scheduleTick := time.NewTicker(c.ScheduleInterval)
	reportTick := time.NewTicker(c.ReportInterval)
	for {
		select {
		case <-ctx.Done():
			return nil

		case <-checkTick.C:
			c.checkSum(context.Background())

		case <-scheduleTick.C:
			c.scheduler.schedule()

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
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`SELECT * FROM %s WHERE id in (%d, %d) FOR UPDATE NOWAIT`, tableName, row, row+c.TableSize))
	if err == nil {
		rows.Close()
		return false, nil
	}
	if me, ok := err.(*mysql.MySQLError); !ok || me.Number != 3572 {
		return false, err
	}
	return true, nil
}

// transfer transfers 1 from a row to the next row.
func (c *pipelineClient) transfer(ctx context.Context, row int) {
	var cnt uint64
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		ctx := context.Background()
		tx, err := c.db.Begin()
		if err != nil {
			log.Fatalf("fail to begin a transaction: %v", err)
		}
		// When pipelined pessimistic locking is enabled, returning successful result means the lock request has been proposed.
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET v = v - 1 WHERE id = %d`, tableName, row)); err != nil {
			log.Fatalf("fail to update: %v", err)
		}
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET v = v + 1 WHERE id = %d`, tableName, row+c.TableSize)); err != nil {
			log.Fatalf("fail to update: %v", err)
		}
		// Check whether the lock above is applied successfully.
		// If there are more than one contenders for each row, it may report false positive because the row can be locked by another transaction.
		if locked, err := c.checkKeyIsLocked(ctx, row); !locked || err != nil {
			if c.Strict {
				log.Fatalf("fail to lock: %v", err)
			}
			atomic.AddUint64(&c.lockErr, 1)
		}
		if err := tx.Commit(); err != nil {
			if c.Strict {
				log.Fatalf("fail to commit: %v", err)
			}
			atomic.AddUint64(&c.commitErr, 1)
		}
		cnt++
		if cnt == 10 {
			atomic.AddUint64(&c.total, cnt)
			cnt = 0
		}
	}
}

// checkSum checks the sum of all rows in the table. It should be 0 because every transaction increases
// one row by 1 and decrease one row by 1.
func (c *pipelineClient) checkSum(ctx context.Context) {
	sum := -1
	err := c.db.QueryRowContext(ctx, fmt.Sprintf(`select sum(v) from %s`, tableName)).Scan(&sum)
	if err != nil {
		log.Fatalf("fail to check sum: %v", err)
	}
	if sum != 0 {
		log.Fatalf("sum(%d) is not zero!", sum)
	}
}

func (c *pipelineClient) reportStats() {
	c.reportCount++
	total := atomic.LoadUint64(&c.total)
	totalDiff := total - c.lastTotal
	c.lastTotal = total

	lockErr := atomic.LoadUint64(&c.lockErr)
	lockErrDiff := lockErr - c.lastLockErr
	c.lastLockErr = lockErr

	commitErr := atomic.LoadUint64(&c.commitErr)
	commitErrDiff := commitErr - c.lastCommitErr
	c.lastCommitErr = commitErr

	log.Infof("[%v] tps: %.2f lock_err/s: %.2f commit_err/s: %.2f lock_err/commit_err/total: %d/%d/%d(%.2f%%)",
		time.Duration(c.reportCount*uint64(c.ReportInterval)).String(),
		float64(totalDiff)/c.ReportInterval.Seconds(),
		float64(lockErrDiff)/c.ReportInterval.Seconds(),
		float64(commitErrDiff)/c.ReportInterval.Seconds(),
		lockErr, commitErr, total, float64(lockErr+commitErr)/float64(total)*100,
	)
}

func (c *pipelineClient) reportSummary() {
	lockErr := atomic.LoadUint64(&c.lockErr)
	commitErr := atomic.LoadUint64(&c.commitErr)
	total := atomic.LoadUint64(&c.total)
	log.Infof("Summary: lock_err/commit_err/total: %d/%d/%d(%.2f%%)", lockErr, commitErr, total, float64(lockErr+commitErr)/float64(total)*100)
}
