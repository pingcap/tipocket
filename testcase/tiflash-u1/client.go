package tiflashu1

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Below are the SQL strings for testing tables.
const (
	tableSchema = `CREATE TABLE if not exists tiflash_u1 (
  a varchar(512) DEFAULT NULL,
  b varchar(512) DEFAULT NULL,
  c varchar(512) DEFAULT NULL,
  d varchar(1000) DEFAULT NULL,
  e varchar(1000) DEFAULT NULL,
  f bigint(11) DEFAULT NULL,
  g bigint(11) DEFAULT NULL,
  h double DEFAULT NULL,
  i bigint(11) DEFAULT NULL,
  j bigint(11) DEFAULT NULL,
  k bigint(11) DEFAULT NULL,
  l bigint(11) DEFAULT NULL,
  m bigint(11) DEFAULT NULL,
  n double DEFAULT NULL,
  o bigint(11) DEFAULT NULL,
  p double DEFAULT NULL,
  q bigint(11) DEFAULT NULL,
  r double DEFAULT NULL,
  s double DEFAULT NULL,
  t bigint(11) DEFAULT NULL,
  u bigint(11) DEFAULT NULL,
  v bigint(11) DEFAULT NULL,
  w varchar(100) DEFAULT NULL,
  x varchar(100) DEFAULT NULL,
  y varchar(100) DEFAULT NULL,
  z varchar(100) DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin`
	tableName = `tiflash_u1`
)

// Config is for tiflash-u1 testing.
type Config struct {
	UpdateConcurrency   int
	VerifyConcurrency   int
	InitialRecordNum    int
	InsertBatchCount    int
	Interval            time.Duration
	DbName              string
	TiFlashDataReplicas int
}

type tiflashU1Case struct {
	cfg *Config
	// Stopped is an atomic field.
	stopped int32
	dbConn  *sql.DB
}

// ClientCreator ...
type ClientCreator struct {
	Cfg *Config
}

// Create ...
func (c ClientCreator) Create(_ cluster.ClientNode) core.Client {
	return NewTiFlashU1Case(c.Cfg)
}

// NewTiFlashU1Case ...
func NewTiFlashU1Case(cfg *Config) core.Client {
	return &tiflashU1Case{
		cfg:     cfg,
		stopped: 0,
	}
}

func (c *tiflashU1Case) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	// Only allow the first client to setup the whole test
	if idx != 0 {
		return nil
	}

	log.Info("[tiflashU1Case] starting SetUp")

	var err error
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/%s", node.IP, node.Port, c.cfg.DbName)
	log.Infof("start to init...")
	db, err := util.OpenDB(dsn, 1)

	if err != nil {
		log.Fatalf("[tiflashU1Case] create db client error %v", err)
	}

	if err := c.initialize(ctx, db); err != nil {
		log.Fatalf("[tiflashu1] initial failed %v", err)
	}

	c.dbConn = db
	return nil
}

func (c *tiflashU1Case) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (c *tiflashU1Case) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	if err := c.execute(ctx, c.dbConn); err != nil {
		log.Fatalf("[tiflashu1] return with error %v", err)
	}
	return nil
}

// String implements fmt.Stringer interface.
func (c *tiflashU1Case) String() string {
	return "tiflashU1"
}

func randString(maxLength int) string {
	letterRunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	length := rand.Intn(maxLength) + 1
	b := make([]rune, length)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func randBigInt() int64 {
	return rand.Int63()
}

func randInt(maxValue int) int {
	return rand.Intn(maxValue)
}

func randDouble() float64 {
	return rand.Float64()
}

// SQLBatchLoader ...
type SQLBatchLoader struct {
	insertHint string
	db         *sql.DB
	buf        bytes.Buffer
	count      int
	batchNum   int
}

func newSQLBatchLoader(db *sql.DB, hint string, batchNum int) *SQLBatchLoader {
	return &SQLBatchLoader{
		count:      0,
		insertHint: hint,
		db:         db,
		batchNum:   batchNum,
	}
}

// InsertValue inserts a value, the loader may flush all pending values.
func (b *SQLBatchLoader) InsertValue(query []string) error {
	sep := ", "
	if b.count == 0 {
		b.buf.WriteString(b.insertHint)
		sep = " "
	}
	b.buf.WriteString(sep)
	b.buf.WriteString(query[0])

	b.count++

	if b.count >= b.batchNum {
		return b.Flush()
	}

	return nil
}

// Flush inserts all pending values
func (b *SQLBatchLoader) Flush() error {
	if b.buf.Len() == 0 {
		return nil
	}

	_, err := b.db.Exec(b.buf.String())
	if err != nil {
		panic(err)
	}
	b.count = 0
	b.buf.Reset()

	return nil
}

func getRow() string {
	return fmt.Sprintf("('%s','%s','%s','%s','%s',%d,%d,%f,%d,%d,%d,%d,%d,%f,%d,%f,%d,%f,%f,%d,%d,%d,'%s','%s','%s','%s') ",
		randString(512),
		randString(512),
		randString(512),
		randString(1000),
		randString(1000),
		randBigInt(),
		randInt(10000),
		randDouble(),
		randBigInt(),
		randBigInt(),
		randBigInt(),
		randBigInt(),
		randBigInt(),
		randDouble(),
		randBigInt(),
		randDouble(),
		randBigInt(),
		randDouble(),
		randDouble(),
		randBigInt(),
		randBigInt(),
		randBigInt(),
		randString(100),
		randString(100),
		randString(100),
		randString(100))
}

func (c *tiflashU1Case) stableUpdateTable(db *sql.DB) {
	loader := newSQLBatchLoader(db, fmt.Sprintf("INSERT INTO %s VALUES ", tableName), c.cfg.InsertBatchCount)
	insertNum := rand.Intn(500)
	deleteNum := rand.Intn(500)
	for i := 0; i < insertNum; i += 1 {
		v := getRow()
		if err := loader.InsertValue([]string{v}); err != nil {
			log.Warn(err)
		}
	}
	if err := loader.Flush(); err != nil {
		log.Warn(err)
	}
	if _, err := db.Exec(fmt.Sprintf("DELETE FROM %s limit %d", tableName, deleteNum)); err != nil {
		log.Warn(err)
	}
}

func (c *tiflashU1Case) initialize(ctx context.Context, db *sql.DB) error {
	if _, err := db.Query(fmt.Sprintf("Drop table if exists %s", tableName)); err != nil {
		return errors.Trace(err)
	}

	if _, err := db.Query(tableSchema); err != nil {
		return errors.Trace(err)
	}

	maxSecondsBeforeTiFlashAvail := 1000
	if c.cfg.TiFlashDataReplicas > 0 {
		if err := util.SetAndWaitTiFlashReplica(ctx, db, c.cfg.DbName, tableName, c.cfg.TiFlashDataReplicas, maxSecondsBeforeTiFlashAvail); err != nil {
			return errors.Trace(err)
		}
	}

	loader := newSQLBatchLoader(db, fmt.Sprintf("INSERT INTO %s VALUES ", tableName), c.cfg.InsertBatchCount)
	for i := 0; i < c.cfg.InitialRecordNum; i += 1 {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if i != 0 && i%10000 == 0 {
			log.Infof("finish insert %d rows", i)
		}
		v := getRow()
		if err := loader.InsertValue([]string{v}); err != nil {
			return errors.Trace(err)
		}
	}
	if err := loader.Flush(); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *tiflashU1Case) execute(ctx context.Context, db *sql.DB) error {
	log.Infof("[%s] start to test...", c)
	defer func() {
		log.Infof("[%s] test end...", c)
	}()

	var wg sync.WaitGroup
	for i := 0; i < c.cfg.VerifyConcurrency; i += 1 {
		wg.Add(1)
		c.startVerify(ctx, &wg, db, i)
	}

	run := func(f func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if atomic.LoadInt32(&c.stopped) != 0 {
					// too many log print in here if return error
					log.Errorf("[%s] tiflashu1 stopped", c)
					return
				}
				f()
			}
		}()
	}

	for i := 0; i < c.cfg.UpdateConcurrency; i++ {
		run(func() { c.stableUpdateTable(db) })
	}

	wg.Wait()
	return nil
}

func (c *tiflashU1Case) checkConsistency(tx *sql.Tx, threadID int, tso uint64, query string) bool {
	var totalTiFlash = -1
	var totalTiKV = -1
	var meetError = false
	if _, err := tx.Query("set @@session.tidb_isolation_read_engines='tikv'"); err != nil {
		log.Warn(err)
	}
	if err := tx.QueryRow(query).Scan(&totalTiKV); err != nil {
		tx.Rollback()
		log.Warn(err)
		meetError = true
	}

	if !meetError {
		if _, err := tx.Query("set @@session.tidb_isolation_read_engines='tiflash'"); err != nil {
			log.Warn(err)
		}
		if err := tx.QueryRow(query).Scan(&totalTiFlash); err != nil {
			tx.Rollback()
			log.Warn(err)
			meetError = true
		}
	}

	if !meetError && totalTiFlash != totalTiKV {
		log.Infof("tiflash result %d, tikv result %d is not consistent thread %d tso %d.\n", totalTiFlash, totalTiKV, threadID, tso)
		return false
	}
	log.Infof("tiflash result %d, tikv result %d thread %d tso %d\n", totalTiFlash, totalTiKV, threadID, tso)
	if !meetError {
		tx.Commit()
	}
	return true
}

func (c *tiflashU1Case) verify(db *sql.DB, threadID int) {
	query := fmt.Sprintf("select count(*) from %s", tableName)
	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	var tso uint64
	if err = tx.QueryRow("select @@tidb_current_ts").Scan(&tso); err != nil {
		panic(err)
	}

	if threadID%3 == 0 {
		interval := rand.Intn(5000)
		log.Infof("thread %d sleep %d millisecond\n", threadID, interval)
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}

	correct := c.checkConsistency(tx, threadID, tso, query)
	if !correct {
		log.Warnf("thread %d meet wrong result, try to check again\n", threadID)
		c.checkConsistency(tx, threadID, tso, query)
		log.Fatalf("thread %d meet wrong result\n", threadID)
	}
}

func (c *tiflashU1Case) startVerify(ctx context.Context, wg *sync.WaitGroup, db *sql.DB, index int) {
	run := func(f func()) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			if atomic.LoadInt32(&c.stopped) != 0 {
				log.Errorf("[%s] test stopped", c)
				return
			}
			f()
		}
	}

	go run(func() {
		c.verify(db, index)
	})
}
