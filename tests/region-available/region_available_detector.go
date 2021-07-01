package regionavailable

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

// Config for regionAvailableDetector
type Config struct {
	DBName          string
	TotalRows       int
	Concurrency     int
	MaxResponseTime time.Duration
	SleepDuration   time.Duration
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(maxLen int) string {
	n := rand.Intn(maxLen) + 1
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// regionAvailableDetector simulates a complete record of financial transactions over the
// life of a bank (or other company).
type regionAvailableDetector struct {
	*Config
	db *sql.DB
}

// ClientCreator creates regionAvailableDetector
type ClientCreator struct {
	Cfg *Config
}

// Create ...
func (l ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &regionAvailableDetector{
		Config: l.Cfg,
	}
}

// SetUp ...
func (d *regionAvailableDetector) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	var err error
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/%s", node.IP, node.Port, d.DBName)

	log.Info("start to init...")
	d.db, err = util.OpenDB(dsn, d.Concurrency)
	if err != nil {
		log.Fatalf("[regionAvailableDetector] create db client error %v", err)
	}
	defer func() {
		log.Info("init end...")
	}()

	// Create table
	if _, err := d.db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.region_available "+
		"(id int(10) PRIMARY KEY, pad varchar(255))", d.DBName)); err != nil {
		log.Fatalf("[regionAvailableDetector] create table fail %v", err)
	}
	if _, err := d.db.Exec(fmt.Sprintf("TRUNCATE TABLE %s.region_available", d.DBName)); err != nil {
		log.Fatalf("[regionAvailableDetector] truncate table fail %v", err)
	}

	// Load data
	if err := d.loadData(); err != nil {
		log.Fatalf("[regionAvailableDetector] load data fail %v", err)
	}
	log.Info("[regionAvailableDetector] load data successfully")

	return nil
}

func (d *regionAvailableDetector) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (d *regionAvailableDetector) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	log.Info("start to test...")
	defer func() {
		log.Info("test end...")
	}()
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	for i := 0; i < d.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				stmt := fmt.Sprintf("UPDATE %s.region_available SET pad=\"%s\" WHERE id = %d", d.DBName, randString(255), rand.Intn(d.TotalRows)+1)
				start := time.Now()
				if _, err := d.db.Exec(stmt); err != nil {
					log.Fatal("[regionAvailableDetector] execute sql failed, ", stmt, err)
				}
				end := time.Now()
				if end.Sub(start) > d.MaxResponseTime {
					logStr := fmt.Sprintf("[regionAvailableDetector] query [%s] exceed max response time, takes %v sencods", stmt, end.Sub(start).Seconds())
					log.Fatal(logStr)
				}

				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(d.SleepDuration)
				}
			}
		}()
	}
	wg.Wait()
	cancel()
	return d.db.Close()
}

func (d *regionAvailableDetector) loadData() error {
	insertSQL := fmt.Sprintf("INSERT IGNORE INTO %s.region_available VALUES", d.DBName)
	stmt := insertSQL
	insertedRows := 0
	for i := 0; i < d.TotalRows; i++ {
		stmt += "(" + strconv.Itoa(i+1) + ", \"" + randString(255) + "\")"
		if i > 0 && i%100 == 0 {
			if _, err := d.db.Exec(stmt); err != nil {
				return err
			}
			stmt = insertSQL
			insertedRows = i + 1
		} else if i+1 < d.TotalRows {
			stmt += ","
		}
	}
	if insertedRows < d.TotalRows {
		if _, err := d.db.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}
