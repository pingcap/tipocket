package readstress

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/codahale/hdrhistogram"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

// CaseCreator is a creator of a read-stress test
type CaseCreator struct {
	NumRows          int
	LargeConcurrency int
	LargeTimeout     time.Duration
	SmallConcurrency int
	SmallTimeout     time.Duration
	ReplicaRead      string
}

// Create creates a read-stress test client
func (c CaseCreator) Create(node cluster.ClientNode) core.Client {
	return &stressClient{
		numRows:          c.NumRows,
		largeConcurrency: c.LargeConcurrency,
		largeTimeout:     c.LargeTimeout,
		smallConcurrency: c.SmallConcurrency,
		smallTimeout:     c.SmallTimeout,
		replicaRead:      c.ReplicaRead,
	}
}

type stressClient struct {
	numRows          int
	largeConcurrency int
	largeTimeout     time.Duration
	smallConcurrency int
	smallTimeout     time.Duration
	db               *sql.DB
	replicaRead      string
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz")

func (c *stressClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	// only prepare data through the first TiDB
	if idx != 0 {
		return nil
	}

	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/test", node.IP, node.Port)

	log.Info("[stressClient] Initializing...")
	var err error
	concurrency := c.largeConcurrency + c.smallConcurrency
	if concurrency < 128 {
		concurrency = 128
	}
	c.db, err = util.OpenDB(dsn, concurrency)
	if err != nil {
		log.Fatal(err)
	}

	util.RandomlyChangeReplicaRead("read-stress", c.replicaRead, c.db)

	if _, err := c.db.Exec("DROP TABLE IF EXISTS t"); err != nil {
		log.Fatal(err)
	}
	if _, err := c.db.Exec("CREATE TABLE t(id INT PRIMARY KEY, k INT, v varchar(255), KEY (v))"); err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	start := 0
	step := c.numRows / 128
	for i := 0; i < 128; i++ {
		end := start + step
		if i+1 == 128 {
			end = c.numRows
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			if err != nil {
				log.Fatal(err)
			}
			stmt, err := c.db.Prepare("INSERT INTO t VALUES (?, ?, ?)")
			if err != nil {
				log.Fatal(err)
			}
			for ; start < end; start += 256 {
				txn, err := c.db.Begin()
				if err != nil {
					log.Fatal(err)
				}
				txnStmt := txn.Stmt(stmt)
				for id := start; id < start+256 && id < end; id++ {
					if _, err := txnStmt.Exec(id, rand.Intn(64), string(letters[rand.Intn(26)])); err != nil {
						log.Fatal(err)
					}
				}
				if err := txn.Commit(); err != nil {
					log.Fatal(err)
				}
			}

		}(start, end)
		start = end
	}
	if _, err := c.db.Exec("ANALYZE TABLE t"); err != nil {
		log.Fatal(err)
	}
	wg.Wait()
	log.Info("Data prepared")
	return nil
}

func (c *stressClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	// only tear down through the first TiDB
	if idx != 0 {
		return nil
	}
	_, err := c.db.Exec("DROP TABLE IF EXISTS t")
	return err
}

func (c *stressClient) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	var wg sync.WaitGroup
	for i := 0; i < c.largeConcurrency; i++ {
		wg.Add(1)
		go func() {
			finished := false
			for !finished {
				ch := make(chan struct{})
				go func() {
					if _, err := c.db.Exec("SELECT v, COUNT(*) FROM t WHERE k % 10 = 1 GROUP BY v HAVING v > 'b'"); err != nil {
						log.Fatal(err)
					}
					ch <- struct{}{}
				}()
				select {
				case <-ch:
					continue
				case <-time.After(c.largeTimeout):
					log.Fatal("[stressClient] Large query timed out")
				case <-ctx.Done():
					finished = true
				}
			}
			wg.Done()
		}()
	}
	deadline, ok := ctx.Deadline()
	log.Info(deadline, ok)
	for !ok || time.Now().Before(deadline) {
		hist := c.runSmall(ctx, time.Second*10)
		mean, quantile99 := hist.Mean(), hist.ValueAtQuantile(99)
		log.Infof("[stressClient] Small queries in the last minute, mean: %d(us), 99th: %d(us)", int64(mean), quantile99)
		if mean > float64(c.smallTimeout.Microseconds()) {
			log.Fatal("[stressClient] Small query timed out")
		}
	}
	wg.Wait()
	log.Info("[stressClient] Test finished.")
	return nil
}

func (c *stressClient) runSmall(ctx context.Context, duration time.Duration) *hdrhistogram.Histogram {
	var wg sync.WaitGroup
	durCh := make(chan time.Duration)
	deadline := time.Now().Add(duration)
	for i := 0; i < c.smallConcurrency; i++ {
		wg.Add(1)
		go func() {
			for time.Now().Before(deadline) {
				beginInst := time.Now()
				txn, err := c.db.Begin()
				if err != nil {
					log.Fatal(err)
				}
				if _, err := txn.Exec("SELECT v FROM t WHERE id IN (?, ?, ?)", rand.Intn(c.numRows), rand.Intn(c.numRows), rand.Intn(c.numRows)); err != nil {
					log.Fatal(err)
				}
				start := rand.Intn(c.numRows)
				end := start + 50
				if _, err := txn.Exec("SELECT k FROM t WHERE id BETWEEN ? AND ? AND v = 'b'", start, end); err != nil {
					log.Fatal(err)
				}
				if err := txn.Commit(); err != nil {
					log.Fatal(err)
				}
				dur := time.Now().Sub(beginInst)
				durCh <- dur
			}
			wg.Done()
		}()
	}
	go func() {
		wg.Wait()
		close(durCh)
	}()
	hist := hdrhistogram.New(0, 60000000, 3)
	for dur := range durCh {
		hist.RecordValue(dur.Microseconds())
	}
	return hist
}
