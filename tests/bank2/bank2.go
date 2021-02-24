package bank2

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

const (
	initialBalance    = 1000
	insertConcurrency = 100
	insertBatchSize   = 100
	maxTransfer       = 100
	systemAccountID   = 0
)

var (
	stmtsCreate = []string{
		`CREATE TABLE IF NOT EXISTS bank2_accounts (
		id INT,
		balance INT NOT NULL,
		name VARCHAR(32),
		remark VARCHAR(2048),
		PRIMARY KEY (id),
		UNIQUE INDEX byName (name),
		KEY byBalance (balance)
	);`,
		`CREATE TABLE IF NOT EXISTS bank2_transaction (
		id INT,
		booking_date TIMESTAMP DEFAULT NOW(),
		txn_date TIMESTAMP DEFAULT NOW(),
		txn_ref VARCHAR(32),
		remark VARCHAR(2048),
		PRIMARY KEY (id),
		UNIQUE INDEX byTxnRef (txn_ref)
	);`,
		`CREATE TABLE IF NOT EXISTS bank2_transaction_leg (
		id INT AUTO_INCREMENT,
		account_id INT,
		amount INT NOT NULL,
		running_balance INT NOT NULL,
		txn_id INT,
		remark VARCHAR(2048),
		PRIMARY KEY (id)
	);`,
		`TRUNCATE TABLE bank2_accounts;`,
		`TRUNCATE TABLE bank2_transaction;`,
		`TRUNCATE TABLE bank2_transaction_leg;`,
	}
	remark = strings.Repeat("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXVZabcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXVZlkjsanksqiszndqpijdslnnq", 16)

	// PadLength returns a random padding length for `remark` field within user
	// specified bound.
	baseLen      = [3]int{36, 48, 16}
	tidbDatabase = true
)

// Config ...
type Config struct {
	// NumAccounts is total accounts
	NumAccounts   int           `toml:"num_accounts"`
	Interval      time.Duration `toml:"interval"`
	Concurrency   int           `toml:"concurrency"`
	RetryLimit    int           `toml:"retry_limit"`
	EnableLongTxn bool          `toml:"enable_long_txn"`
	Contention    string        `toml:"contention"`
	Pessimistic   bool          `toml:"pessimistic"`
	MinLength     int           `toml:"min_length"`
	MaxLength     int           `toml:"max_length"`
	ReplicaRead   string
	DbName        string
}

// ClientCreator ...
type ClientCreator struct {
	Cfg *Config
}

type bank2Client struct {
	*Config
	wg    sync.WaitGroup
	stop  int32
	txnID int32
	db    *sql.DB
}

func (c *bank2Client) padLength(table int) int {
	minLen := 0
	maxLen := len(remark)
	if table >= 0 && table < 3 {
		if c.Config.MinLength > baseLen[table] {
			minLen = c.Config.MinLength - baseLen[table]
		}
		if c.Config.MaxLength < maxLen+baseLen[table] {
			maxLen = c.Config.MaxLength - baseLen[table]
		}
	}
	return minLen + rand.Intn(maxLen-minLen)
}

func (c *bank2Client) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	var err error
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/%s", node.IP, node.Port, c.DbName)

	log.Infof("start to init...")
	db, err := util.OpenDB(dsn, 1)
	if err != nil {
		log.Fatalf("[bank2Client] create db client error %v", err)
	}
	util.RandomlyChangeReplicaRead(c.String(), c.ReplicaRead, db)

	_, err = db.Exec("set @@global.tidb_txn_mode = 'pessimistic';")
	if err != nil {
		log.Fatalf("[bank2Client] set txn_mode failed: %v", err)
	}
	time.Sleep(5 * time.Second)
	c.db, err = util.OpenDB(dsn, c.Concurrency)
	util.RandomlyChangeReplicaRead(c.String(), c.ReplicaRead, c.db)

	if err != nil {
		return err
	}
	defer func() {
		log.Infof("init end...")
	}()

	for _, stmt := range stmtsCreate {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("execute statement %s error %v", stmt, err)
		}
	}

	var wg sync.WaitGroup
	type Job struct {
		begin, end int
	}

	ch := make(chan Job)
	for i := 0; i < insertConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range ch {
				select {
				case <-ctx.Done():
					return
				default:
				}
				start := time.Now()
				args := make([]string, 0, insertBatchSize)
				for i := job.begin; i < job.end; i++ {
					args = append(args, fmt.Sprintf(`(%d, %d, "account %d", "%s")`, i, initialBalance, i, remark[:c.padLength(0)]))
				}

				query := fmt.Sprintf("INSERT IGNORE INTO bank2_accounts (id, balance, name, remark) VALUES %s", strings.Join(args, ","))
				err := util.RunWithRetry(ctx, c.Config.RetryLimit, 5*time.Second, func() error {
					_, err := db.Exec(query)
					if util.IsErrDupEntry(err) {
						return nil
					}
					return err
				})
				if err != nil {
					log.Fatalf("[%s] exec %s err %v", c, query, err)
				}
				log.Infof("[%s] insert %d accounts, takes %s", c, job.end-job.begin, time.Since(start))
			}
		}()
	}

	var begin, end int
	for begin = 1; begin <= c.Config.NumAccounts; begin = end {
		end = begin + insertBatchSize
		if end > c.Config.NumAccounts {
			end = c.Config.NumAccounts + 1
		}
		select {
		case <-ctx.Done():
			return nil
		case ch <- Job{begin: begin, end: end}:
		}
	}
	close(ch)
	wg.Wait()

	select {
	case <-ctx.Done():
		log.Warnf("[%s] initialize is cancelled", c)
		return nil
	default:
	}
	query := fmt.Sprintf(`INSERT IGNORE INTO bank2_accounts (id, balance, name) VALUES (%d, %d, "system account")`, systemAccountID, int64(c.Config.NumAccounts*initialBalance))
	err = util.RunWithRetry(ctx, c.Config.RetryLimit, 3*time.Second, func() error {
		_, e := db.Exec(query)
		if util.IsErrDupEntry(e) {
			return nil
		}
		return e
	})
	if err != nil {
		log.Fatalf("[%s] insert system account err: %v", c, err)
	}

	c.startVerify(ctx, db)
	return nil
}

func (c *bank2Client) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (c *bank2Client) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	log.Infof("[%s] start to test...", c)
	defer func() {
		log.Infof("[%s] test end...", c)
	}()
	var wg sync.WaitGroup
	for i := 0; i < c.Config.Concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if atomic.LoadInt32(&c.stop) != 0 {
					log.Errorf("[%s] stopped", c)
					return
				}
				c.wg.Add(1)
				c.moveMoney(c.db)
				c.wg.Done()
			}
		}(i)
	}
	wg.Wait()
	return nil
}

func (c *bank2Client) startVerify(ctx context.Context, db *sql.DB) {
	c.verify(db)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.Config.Interval):
				c.verify(db)
			}
		}
	}()
}

func (c *bank2Client) verify(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		_ = errors.Trace(err)
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if tidbDatabase {
		var tso uint64
		if err = tx.QueryRow("SELECT @@tidb_current_ts").Scan(&tso); err == nil {
			log.Infof("[%s] SELECT SUM(balance) to verify use tso %d", c, tso)
		}
	}

	var total int64
	expectTotal := (int64(c.Config.NumAccounts) * initialBalance) * 2

	// query with IndexScan
	uuid := fastuuid.MustNewGenerator().Hex128()
	query := fmt.Sprintf("SELECT SUM(balance) AS total, '%s' as uuid FROM bank2_accounts", uuid)
	if err = tx.QueryRow(query).Scan(&total, &uuid); err != nil {
		_ = errors.Trace(err)
		return
	}
	if total != expectTotal {
		log.Errorf("[%s] bank2_accounts total should be %d, but got %d, query uuid is %s", c, expectTotal, total, uuid)
		atomic.StoreInt32(&c.stop, 1)
		c.wg.Wait()
		log.Fatalf("[%s] bank2_accounts total should be %d, but got %d, query uuid is %s", c, expectTotal, total, uuid)
	}

	// query with TableScan
	uuid = fastuuid.MustNewGenerator().Hex128()
	query = fmt.Sprintf("SELECT SUM(balance) AS total, '%s' as uuid FROM bank2_accounts ignore index(byBalance)", uuid)
	if err = tx.QueryRow(query).Scan(&total, &uuid); err != nil {
		_ = errors.Trace(err)
		return
	}
	if total != expectTotal {
		log.Errorf("[%s] bank2_accounts total should be %d, but got %d, query uuid is %s", c, expectTotal, total, uuid)
		atomic.StoreInt32(&c.stop, 1)
		c.wg.Wait()
		log.Fatalf("[%s] bank2_accounts total should be %d, but got %d, query uuid is %s", c, expectTotal, total, uuid)
	}

	if _, err := tx.Exec("ADMIN CHECK TABLE bank2_accounts"); err != nil {
		log.Errorf("[%s] ADMIN CHECK TABLE bank2_accounts fails: %v", c, err)
		atomic.StoreInt32(&c.stop, 1)
		c.wg.Wait()
		log.Fatalf("[%s] ADMIN CHECK TABLE bank2_accounts fails: %v", c, err)
	}
}

func (c *bank2Client) moveMoney(db *sql.DB) {
	from, to := rand.Intn(c.Config.NumAccounts), rand.Intn(c.Config.NumAccounts)
	if c.Config.Contention == "high" {
		// Use the first account number we generated as a coin flip to
		// determine whether we're transferring money into or out of
		// the system account.
		if from > c.Config.NumAccounts/2 {
			from = systemAccountID
		} else {
			to = systemAccountID
		}
	}
	if from == to {
		return
	}
	amount := rand.Intn(maxTransfer)
	// start := time.Now()
	if err := c.execTransaction(db, from, to, amount); err != nil {
		// bank2VerifyFailedCounter.Inc()
		log.Errorf("[%s] move money err %v", c, err)
		return
	}
	// bank2VerifyDuration.Observe(time.Since(start).Seconds())
}

func (c *bank2Client) execTransaction(db *sql.DB, from, to int, amount int) error {
	tx, err := db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	rows, err := tx.Query(fmt.Sprintf("SELECT id, balance FROM bank2_accounts WHERE id IN (%d, %d) FOR UPDATE", from, to))
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

	if fromBalance < amount {
		return nil
	}

	insertTxn := `INSERT INTO bank2_transaction (id, txn_ref, remark) VALUES (?, ?, ?)`
	insertTxnLeg := `INSERT INTO bank2_transaction_leg (account_id, amount, running_balance, txn_id, remark) VALUES (?, ?, ?, ?, ?)`
	updateAcct := `UPDATE bank2_accounts SET balance = ? WHERE id = ?`
	txnID := atomic.AddInt32(&c.txnID, 1)
	if _, err := tx.Exec(insertTxn, txnID, fmt.Sprintf("txn %d", txnID), remark[:c.padLength(1)]); err != nil {
		_ = tx.Rollback()
		return errors.Trace(err)
	}
	if _, err = tx.Exec(insertTxnLeg, from, -amount, fromBalance-amount, txnID, remark[:c.padLength(2)]); err != nil {
		_ = tx.Rollback()
		return errors.Trace(err)
	}
	if _, err = tx.Exec(insertTxnLeg, to, amount, toBalance+amount, txnID, remark[:c.padLength(2)]); err != nil {
		_ = tx.Rollback()
		return errors.Trace(err)
	}
	if _, err = tx.Exec(updateAcct, toBalance+amount, to); err != nil {
		_ = tx.Rollback()
		return errors.Trace(err)
	}
	if _, err = tx.Exec(updateAcct, fromBalance-amount, from); err != nil {
		_ = tx.Rollback()
		return errors.Trace(err)
	}

	err = tx.Commit()
	if err != nil {
		log.Infof("[%s] %d(%d) failed to transfer %d to %d(%d), err: %v", c, from, fromBalance, amount, to, toBalance, err)
	} else {
		log.Infof("[%s] %d(%d) transfer %d to %d(%d)", c, from, fromBalance, amount, to, toBalance)
	}
	return nil
}

// Create ...
func (c ClientCreator) Create(_ cluster.ClientNode) core.Client {
	return &bank2Client{
		Config: c.Cfg,
	}
}

// String ...
func (c *bank2Client) String() string {
	return "bank2"
}
