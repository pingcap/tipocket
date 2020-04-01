package readstress

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/util"

	"github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
)

// CaseCreator is a creator of a read-stress test
type CaseCreator struct {
	NumRows          int
	LargeConcurrency int
	LargeTimeout     time.Duration
	SmallConcurrency int
	SmallTimeout     time.Duration
}

// Create creates a read-stress test client
func (c CaseCreator) Create(node types.ClientNode) core.Client {
	return &stressClient{
		numRows:          c.NumRows,
		largeConcurrency: c.LargeConcurrency,
		largeTimeout:     c.LargeTimeout,
		smallConcurrency: c.SmallConcurrency,
		smallTimeout:     c.SmallTimeout,
	}
}

type stressClient struct {
	numRows          int
	largeConcurrency int
	largeTimeout     time.Duration
	smallConcurrency int
	smallTimeout     time.Duration
	db               *sql.DB
}

func (c *stressClient) SetUp(ctx context.Context, nodes []types.ClientNode, idx int) error {
	// only prepare data through the first TiDB
	if idx != 0 {
		return nil
	}

	node := nodes[idx]
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
			for ; start < end; start += 256 {
				txn, err := c.db.Begin()
				if err != nil {
					log.Fatal(err)
				}
				for id := start; id < start+256 && id < end; id++ {
					if _, err := txn.Exec("INSERT INTO t VALUES (?, ?, ?)", id, rand.Intn(64), string(rand.Intn(26)+'a')); err != nil {
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

func (c *stressClient) TearDown(ctx context.Context, nodes []types.ClientNode, idx int) error {
	// only tear down through the first TiDB
	if idx != 0 {
		return nil
	}
	_, err := c.db.Exec("DROP TABLE IF EXISTS t")
	return err
}

func (c *stressClient) Invoke(ctx context.Context, node types.ClientNode, r interface{}) interface{} {
	panic("implement me")
}

func (c *stressClient) NextRequest() interface{} {
	panic("implement me")
}

func (c *stressClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("implement me")
}

func (c *stressClient) Start(ctx context.Context, cfg interface{}, clientNodes []types.ClientNode) error {
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
	for i := 0; i < c.smallConcurrency; i++ {
		wg.Add(1)
		go func() {
			finished := false
			for !finished {
				ch := make(chan struct{})
				go func() {
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
					ch <- struct{}{}
				}()
				select {
				case <-ch:
					continue
				case <-time.After(c.smallTimeout):
					log.Fatal("[stressClient] Small query timed out")
				case <-ctx.Done():
					finished = true
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
	log.Info("[stressClient] Test finished.")
	return nil
}
