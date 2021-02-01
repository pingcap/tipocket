package bank

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"github.com/rogpeppe/fastuuid"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	defaultVerifyTimeout = 6 * time.Hour
	remark               = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXVZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXVZlkjsanksqiszndqpijdslnnq"
)

var (
	// Note: This field is always true in tidb testing.
	tidbDatabase = true
)

type delayMode = int

const (
	noDelay delayMode = iota
	delayRead
	delayCommit
)

const (
	minDelayDuration = time.Minute*10 - time.Second*10
	maxDelayDuration = time.Minute*10 + time.Second*10
)

// Below are the SQL strings for testing tables.
const (
	dropAccountTableTemplate   = `drop table if exists accounts%s`
	dropRecordTableTemplate    = `drop table if exists record`
	createAccountTableTemplate = `create table if not exists accounts%s (id BIGINT PRIMARY KEY, balance BIGINT NOT NULL, remark VARCHAR(128))`
	createRecordTableTemplate  = `create table if not exists record (id BIGINT AUTO_INCREMENT,
from_id BIGINT NOT NULL,
to_id BIGINT NOT NULL,
from_balance BIGINT NOT NULL,
to_balance BIGINT NOT NULL,
amount BIGINT NOT NULL,
tso BIGINT UNSIGNED NOT NULL,
PRIMARY KEY(id))`
)

// Config is for bank testing.
type Config struct {
	EnableLongTxn bool
	Pessimistic   bool
	RetryLimit    int
	Accounts      int
	Tables        int
	Interval      time.Duration
	Concurrency   int
	ReplicaRead   string
	DbName        string
}

// BankCase is for concurrent balance transfer.
type bankCase struct {
	mu  sync.RWMutex
	cfg *Config
	wg  sync.WaitGroup
	// Stopped is an atomic field.
	stopped int32

	dbConn *sql.DB
}

// ClientCreator ...
type ClientCreator struct {
	Cfg *Config
}

// Create ...
func (c ClientCreator) Create(_ cluster.ClientNode) core.Client {
	return NewBankCase(c.Cfg)
}

// NewBankCase ...
func NewBankCase(cfg *Config) core.Client {
	if cfg.Tables < 1 {
		cfg.Tables = 1
	}
	return &bankCase{
		mu:      sync.RWMutex{},
		cfg:     cfg,
		wg:      sync.WaitGroup{},
		stopped: 0,
	}
}

func (c *bankCase) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	// Only allow the first client to setup the whole test
	if idx != 0 {
		return nil
	}

	log.Info("[bankCase] starting SetUp")

	var err error
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/%s", node.IP, node.Port, c.cfg.DbName)
	log.Infof("start to init...")
	db, err := util.OpenDB(dsn, 1)

	if err != nil {
		log.Fatalf("[bankCase] create db client error %v", err)
	}

	if err := c.Initialize(ctx, db); err != nil {
		log.Fatalf("[bank] initial failed %v", err)
	}

	util.RandomlyChangeReplicaRead(c.String(), c.cfg.ReplicaRead, db)

	c.dbConn = db
	return nil
}

func (c *bankCase) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (c *bankCase) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	if err := c.Execute(ctx, c.dbConn); err != nil {
		log.Fatalf("[bank] return with error %v", err)
	}
	return nil
}

/* Below are the parts ported from Schrodinger */

// tryDrop try to drop the database.
// It returns if the database with the index exists.
func (c *bankCase) tryDrop(db *sql.DB, index int) bool {
	var (
		count int
		table string
	)
	//if table is not exist ,return true directly
	query := fmt.Sprintf("show tables like 'accounts%d'", index)
	err := db.QueryRow(query).Scan(&table)
	switch {
	case err == sql.ErrNoRows:
		return true
	case err != nil:
		log.Fatalf("[%s] execute query %s error %v", c, query, err)
	}

	query = fmt.Sprintf("select count(*) as count from accounts%d", index)
	err = db.QueryRow(query).Scan(&count)
	if err != nil {
		log.Fatalf("[%s] execute query %s error %v", c, query, err)
	}
	if count == c.cfg.Accounts {
		return false
	}

	log.Infof("[%s] we need %d accounts%d but got %d, re-initialize the data again", c, c.cfg.Accounts, index, count)

	util.MustExec(db, fmt.Sprintf("drop table if exists accounts%d", index))
	util.MustExec(db, "DROP TABLE IF EXISTS record")
	return true
}

func (c *bankCase) verify(ctx context.Context, db *sql.DB, index string, delay delayMode) error {
	var total int

	tx, err := db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	defer tx.Rollback()

	if delay == delayRead {
		err = c.delay(ctx)
		if err != nil {
			return err
		}
	}

	uuid := fastuuid.MustNewGenerator().Hex128()
	query := fmt.Sprintf("select sum(balance) as total, '%s' as uuid from accounts%s", uuid, index)
	if err = tx.QueryRow(query).Scan(&total, &uuid); err != nil {
		log.Errorf("[%s] select sum error %v", c, err)
		return errors.Trace(err)
	}
	if tidbDatabase {
		var tso uint64
		if err = tx.QueryRow("select @@tidb_current_ts").Scan(&tso); err != nil {
			return errors.Trace(err)
		}
		log.Infof("[%s] select sum(balance) to verify use tso %d", c, tso)
	}
	tx.Commit()
	check := c.cfg.Accounts * 1000
	if total != check {
		log.Errorf("[%s] accouts%s total must %d, but got %d, query uuid is %s", c, index, check, total, uuid)
		atomic.StoreInt32(&c.stopped, 1)
		c.wg.Wait()
		log.Fatalf("[%s] accouts%s total must %d, but got %d, query uuid is %s", c, index, check, total, uuid)
	}

	return nil
}

func (c *bankCase) delay(ctx context.Context) error {
	start := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	delayDuration := minDelayDuration + time.Duration(rand.Int63n(int64(maxDelayDuration-minDelayDuration)))
	for {
		select {
		case <-ctx.Done():
			return errors.New("context canceled")
		case <-ticker.C:
			if atomic.LoadInt32(&c.stopped) != 0 {
				return errors.New("stopped")
			}
			if time.Since(start) > delayDuration {
				return nil
			}
		}
	}
}

func (c *bankCase) startVerify(ctx context.Context, db *sql.DB, index string) {
	c.verify(ctx, db, index, noDelay)

	run := func(f func()) {
		ticker := time.NewTicker(c.cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				f()
			}
		}
	}

	start := time.Now()
	go run(func() {
		err := c.verify(ctx, db, index, noDelay)
		if err != nil {
			log.Infof("[%s] verify error: %s in: %s", c, err, time.Now())
			if time.Now().Sub(start) > defaultVerifyTimeout {
				atomic.StoreInt32(&c.stopped, 1)
				log.Infof("[%s] stop bank execute", c)
				c.wg.Wait()
				log.Fatalf("[%s] verify timeout since %s, error: %s", c, start, err)
			}
		} else {
			start = time.Now()
			log.Infof("[%s] verify success in %s", c, time.Now())
		}
	})

	if c.cfg.EnableLongTxn {
		go run(func() { c.verify(ctx, db, index, delayRead) })
	}
}

func (c *bankCase) initDB(ctx context.Context, db *sql.DB, id int) error {
	var index string
	if id > 0 {
		index = fmt.Sprintf("%d", id)
	}
	isDropped := c.tryDrop(db, id)
	if !isDropped {
		c.startVerify(ctx, db, index)
		return nil
	}

	util.MustExec(db, fmt.Sprintf(dropAccountTableTemplate, index))
	util.MustExec(db, dropRecordTableTemplate)
	util.MustExec(db, fmt.Sprintf(createAccountTableTemplate, index))
	util.MustExec(db, createRecordTableTemplate)
	var wg sync.WaitGroup

	// TODO: fix the error is NumAccounts can't be divided by batchSize.
	// Insert batchSize values in one SQL.
	batchSize := 100
	jobCount := c.cfg.Accounts / batchSize

	maxLen := len(remark)
	ch := make(chan int, jobCount)
	for i := 0; i < c.cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			args := make([]string, batchSize)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				startIndex, ok := <-ch
				if !ok {
					break
				}
				start := time.Now()
				for i := 0; i < batchSize; i++ {
					args[i] = fmt.Sprintf("(%d, %d, \"%s\")", startIndex+i, 1000, remark[:rand.Intn(maxLen)])
				}

				query := fmt.Sprintf("INSERT IGNORE INTO accounts%s (id, balance, remark) VALUES %s", index, strings.Join(args, ","))
				insertF := func() error {
					_, err := db.Exec(query)
					if util.IsErrDupEntry(err) {
						return nil
					}
					return err
				}
				err := util.RunWithRetry(ctx, c.cfg.RetryLimit, 5*time.Second, insertF)
				if err != nil {
					log.Fatalf("[%s]exec %s  err %s", c, query, err)
				}
				log.Infof("[%s] insert %d accounts%s, takes %s", c, batchSize, index, time.Now().Sub(start))
			}
		}()
	}

	for i := 0; i < jobCount; i++ {
		ch <- i * batchSize
	}

	close(ch)
	wg.Wait()

	select {
	case <-ctx.Done():
		log.Warn("[%s] bank initialize is cancel", c)
		return nil
	default:
	}

	c.startVerify(ctx, db, index)
	return nil
}

// Execute implements Case Execute interface.
func (c *bankCase) Execute(ctx context.Context, db *sql.DB) error {
	log.Infof("[%s] start to test...", c)
	defer func() {
		log.Infof("[%s] test end...", c)
	}()
	var wg sync.WaitGroup

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
					log.Errorf("[%s] bank stopped", c)
					return
				}
				c.wg.Add(1)
				f()
				c.wg.Done()
			}
		}()
	}

	for i := 0; i < c.cfg.Concurrency; i++ {
		run(func() { c.moveMoney(ctx, db, noDelay) })
	}
	if c.cfg.EnableLongTxn {
		run(func() { c.moveMoney(ctx, db, delayRead) })
		run(func() { c.moveMoney(ctx, db, delayCommit) })
	}

	wg.Wait()
	return nil
}

// String implements fmt.Stringer interface.
func (c *bankCase) String() string {
	return "bank"
}

func (c *bankCase) moveMoney(ctx context.Context, db *sql.DB, delay delayMode) {
	var (
		from, to, id int
		index        string
	)
	for {
		from, to, id = rand.Intn(c.cfg.Accounts), rand.Intn(c.cfg.Accounts), rand.Intn(c.cfg.Tables)
		if from == to {
			continue
		}
		break
	}
	if id > 0 {
		index = fmt.Sprintf("%d", id)
	}

	amount := rand.Intn(999)

	err := c.execTransaction(ctx, db, from, to, amount, index, delay)

	if err != nil {
		return
	}
}

func (c *bankCase) execTransaction(ctx context.Context, db *sql.DB, from, to int, amount int, index string, delay delayMode) error {
	tx, err := db.Begin()
	if err != nil {
		return errors.Trace(err)
	}

	defer tx.Rollback()

	if delay == delayRead {
		err = c.delay(ctx)
		if err != nil {
			return err
		}
	}

	rows, err := tx.Query(fmt.Sprintf("SELECT id, balance FROM accounts%s WHERE id IN (%d, %d) FOR UPDATE", index, from, to))
	if err != nil {
		return errors.Trace(err)
	}
	defer rows.Close()

	var (
		fromBalance int
		toBalance   int
		count       int
	)

	for rows.Next() {
		var id, balance int
		if err = rows.Scan(&id, &balance); err != nil {
			return errors.Trace(err)
		}
		switch id {
		case from:
			fromBalance = balance
		case to:
			toBalance = balance
		default:
			log.Fatalf("[%s] got unexpected account %d", c, id)
		}

		count++
	}

	if err = rows.Err(); err != nil {
		return errors.Trace(err)
	}

	if count != 2 {
		log.Fatalf("[%s] select %d(%d) -> %d(%d) invalid count %d", c, from, fromBalance, to, toBalance, count)
	}

	var update string
	if fromBalance >= amount {
		update = fmt.Sprintf(`
UPDATE accounts%s
  SET balance = CASE id WHEN %d THEN %d WHEN %d THEN %d END
  WHERE id IN (%d, %d)
`, index, to, toBalance+amount, from, fromBalance-amount, from, to)
		_, err = tx.Exec(update)
		if err != nil {
			return errors.Trace(err)
		}

		var tso uint64
		if tidbDatabase {
			if err = tx.QueryRow("select @@tidb_current_ts").Scan(&tso); err != nil {
				return err
			}
		} else {
			tso = uint64(time.Now().UnixNano())
		}
		if _, err = tx.Exec(fmt.Sprintf(`
INSERT INTO record (from_id, to_id, from_balance, to_balance, amount, tso)
    VALUES (%d, %d, %d, %d, %d, %d)`, from, to, fromBalance, toBalance, amount, tso)); err != nil {
			return err
		}
		log.Infof("[%s] exec pre: %s", c, update)
	}

	if delay == delayCommit {
		err = c.delay(ctx)
		if err != nil {
			return err
		}
	}

	err = tx.Commit()
	if fromBalance >= amount {
		if err != nil {
			log.Infof("[%s] exec commit error: %s\n err:%s", c, update, err)
		}
		if err == nil {
			log.Infof("[%s] exec commit success: %s", c, update)
		}
	}
	return err
}

// Initialize implements Case Initialize interface.
func (c *bankCase) Initialize(ctx context.Context, db *sql.DB) error {
	log.Infof("[%s] start to init...", c)
	defer func() {
		log.Infof("[%s] init end...", c)
	}()
	for i := 0; i < c.cfg.Tables; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		err := c.initDB(ctx, db, i)
		if err != nil {
			return err
		}
	}
	return nil
}
