package follower

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

// ClientCreator creates follower client
type ClientCreator struct {
	Cfg *Config
}

// Config for follower read test
type Config struct {
	DBName           string
	Concurrency      int
	Switch           bool
	SeqLoop          int
	SplitRegionRange int
	InsertNum        int
	EnableSplit      bool
}

type follower struct {
	*Config
	db *sql.DB
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Create creates FollowerReadClient
func (l ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &follower{
		Config: l.Cfg,
	}
}

// SetUp
func (f *follower) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	var err error
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/%s", node.IP, node.Port, f.DBName)

	log.Infof("start to init...")
	f.db, err = util.OpenDB(dsn, f.Concurrency)
	if err != nil {
		log.Fatal(err)
	}

	time.Sleep(2 * time.Second)
	return nil
}

// TearDown
func (f *follower) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (f *follower) ScheduledClientExtensions() core.OnScheduleClientExtensions {
	panic("implement me")
}

func (f *follower) AutoDriveClientExtensions() core.AutoDriveClientExtensions {
	return f
}

// Start
func (f *follower) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	log.Info("start to test...")

	log.Info("testSwitchFollowerRead")
	testSwitchFollowerRead(f)
	log.Info("testInvalidSet")
	testInvalidSet(f)
	log.Info("testValidSet")
	testValidSet(f)
	log.Info("testSession")
	testSession(ctx, f)
	log.Info("testCorrectnes")
	testCorrectnes(f)
	log.Info("testGlobal")
	testGlobal(f)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("testSplitRegion")
		testSplitRegion(f)
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info("testSequence")
		testSequence(ctx, f)
	}()
	wg.Wait()

	err := f.db.Close()
	if err != nil {
		return err
	}

	return nil
}

func testSwitchFollowerRead(f *follower) {
	rows, err := f.db.Query("select @@tidb_replica_read")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var val string
		rows.Scan(&val)
		if val != "leader" {
			log.Fatalf("assert tidb_replica_read == leader failed")
		}
	}
}

func testInvalidSet(f *follower) {
	setVal := []string{"0", "leaner", "adf"}
	for _, val := range setVal {
		_, err := f.db.Query(fmt.Sprintf("set @@tidb_replica_read = \"%v\"", val))
		if err == nil {
			log.Fatalf("set @@tidb_replica_read = %v succeed", val)
		}
	}
}

func testValidSet(f *follower) {
	setVal := []string{"null", "follower", "leader-and-follower", "leader"}
	for _, val := range setVal {
		var sql string
		if val != "null" {
			sql = fmt.Sprintf("set @@tidb_replica_read = \"%v\"", val)
		} else {
			sql = fmt.Sprintf("set @@tidb_replica_read = null")
		}
		_, err := f.db.Query(sql)
		if err != nil {
			log.Fatalf("set @@tidb_replica_read = %v failed: %v", val, err)
		}
	}
}

func testSession(ctx context.Context, f *follower) {
	sql0 := "set @@tidb_replica_read = \"follower\""
	sql1 := "set @@tidb_replica_read = \"leader\""
	sql2 := "set @@tidb_replica_read = \"leader-and-follower\""

	c0, _ := f.db.Conn(ctx)
	c0.ExecContext(ctx, sql0)
	c1, _ := f.db.Conn(ctx)
	c1.ExecContext(ctx, sql1)
	c2, _ := f.db.Conn(ctx)
	c2.ExecContext(ctx, sql2)

	var replicaRead string
	if err := c2.QueryRowContext(ctx, "select @@tidb_replica_read").Scan(&replicaRead); err != nil {
		log.Fatalf("get tidb_replica_read fail: %v", err)
	}
	if replicaRead != "leader-and-follower" {
		log.Fatalf("assert tidb_replica_read == leader-and-follower failed: right is %v", replicaRead)
	}
	if err := c0.QueryRowContext(ctx, "select @@tidb_replica_read").Scan(&replicaRead); err != nil {
		log.Fatalf("get tidb_replica_read fail: %v", err)
	}
	if replicaRead != "follower" {
		log.Fatalf("assert tidb_replica_read == follower failed: right is %v", replicaRead)
	}
	if err := c1.QueryRowContext(ctx, "select @@tidb_replica_read").Scan(&replicaRead); err != nil {
		log.Fatalf("get tidb_replica_read fail: %v", err)
	}
	if replicaRead != "leader" {
		log.Fatalf("assert tidb_replica_read == leader failed: right is %v", replicaRead)
	}

}

func testCorrectnes(f *follower) {
	setVal := []string{"null", "follower", "leader-and-follower", "leader"}
	for i := 0; i < 1000; i++ {
		for _, val := range setVal {
			var sql string
			if val != "null" {
				sql = fmt.Sprintf("set @@tidb_replica_read = \"%v\"", val)
			} else {
				sql = fmt.Sprintf("set @@tidb_replica_read = null")
			}
			rows, err := f.db.Query(sql)
			if err != nil {
				log.Fatalf("set @@tidb_replica_read = %v failed: %v", val, err)
			}
			defer rows.Close()
			for rows.Next() {
				var read string
				rows.Scan(&read)
				if read != val {
					log.Fatalf("assert tidb_replica_read == %v failed", val)
				}
			}
		}
	}
}

func testGlobal(f *follower) {
	_, e1 := f.db.Query("set @@global.tidb_replica_read=follower")
	if e1 == nil {
		log.Fatalf("set @@global.tidb_replica_read=follower succeed")
	}
	_, e2 := f.db.Query("select @@global.tidb_replica_read")
	if e2 == nil {
		log.Fatalf("select @@global.tidb_replica_read succeed")
	}
}

func testSplitRegion(f *follower) {
	region := f.SplitRegionRange
	insertnum := f.InsertNum
	f.db.Exec("create table test_region(a int)")

	if f.Switch {
		_, e := f.db.Exec("set @@tidb_replica_read = \"follower\"")
		if e != nil {
			log.Fatal(e)
		}
		rows, err := f.db.Query("select @@tidb_replica_read")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var val string
			rows.Scan(&val)
			if val != "follower" {
				log.Fatalf("assert tidb_replica_read == follower failed, current: %v", val)
			}
			log.Info("testSplitRegion in follower mode")
		}
	} else {
		_, e := f.db.Exec("set @@tidb_replica_read = \"leader\"")
		if e != nil {
			log.Fatal(e)
		}
		rows, err := f.db.Query("select @@tidb_replica_read")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var val string
			rows.Scan(&val)
			if val != "leader" {
				log.Fatalf("assert tidb_replica_read == leader failed, current: %v", val)
			}
		}
		log.Info("testSplitRegion in leader mode")
	}

	// prepare some data
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < insertnum; i++ {
				_, e := f.db.Exec(fmt.Sprintf("insert into test_region (a) values (%v)", rand.Intn(region)))
				if e != nil {
					log.Fatal(e)
				}
			}
		}()
	}

	// split region
	if f.EnableSplit {
		var regionMax = 300
		log.Info("Enable split")
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Minute)
			for {
				_, e := f.db.Exec(fmt.Sprintf("split table test_region between (0) and (%v) regions %v;", region, regionMax))
				if e != nil {
					log.Fatal(e)
				}
				log.Infof("Split region done")
				regionMax += 50
				if regionMax >= 1000 {
					break
				}
				time.Sleep(1 * time.Minute)
			}
		}()
	}

	// read
	time.Sleep(3 * time.Minute)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				_, e := f.db.Exec("select count(*) from test_region")
				if e != nil {
					log.Fatal(e)
				}
			}
		}()
	}

	wg.Wait()

}

func testSequence(ctx context.Context, f *follower) {
	sql0 := "set @@tidb_replica_read = \"follower\""
	sql1 := "set @@tidb_replica_read = \"leader-and-follower\""
	c0, _ := f.db.Conn(ctx)
	c0.ExecContext(ctx, sql0)
	c0.ExecContext(ctx, "CREATE SEQUENCE seq")
	c1, _ := f.db.Conn(ctx)
	c1.ExecContext(ctx, sql1)

	var mutex sync.Mutex
	m := make(map[int]bool)

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var num int
		for i := 0; i < f.SeqLoop; i++ {
			if err := c0.QueryRowContext(ctx, "SELECT nextval(seq)").Scan(&num); err != nil {
				log.Fatalf("SELECT nextval(seq) fail: %v", err)
			}

			mutex.Lock()
			if m[num] == true {
				log.Fatalf("%v existed", num)
			} else {
				m[num] = true
			}
			mutex.Unlock()

		}
	}()
	go func() {
		defer wg.Done()
		var num int
		for i := 0; i < f.SeqLoop; i++ {
			if err := c1.QueryRowContext(ctx, "SELECT nextval(seq)").Scan(&num); err != nil {
				log.Fatalf("SELECT nextval(seq) fail: %v", err)
			}

			mutex.Lock()
			if m[num] == true {
				log.Fatalf("%v existed", num)
			} else {
				m[num] = true
			}
			mutex.Unlock()

		}
	}()
	wg.Wait()
	log.Infof("testSequence finished, assert sum of sequence succeed")
}
