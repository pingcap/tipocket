package ledger

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

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

var (
	insertConcurrency = 100
	insertBatchSize   = 100
	maxTransfer       = 100
	retryLimit        = 200
)

const (
	stmtDrop   = `DROP TABLE IF EXISTS ledger_accounts`
	stmtCreate = `
CREATE TABLE IF NOT EXISTS ledger_accounts (
  causality_id BIGINT NOT NULL,
  posting_group_id BIGINT NOT NULL,
  amount BIGINT,
  balance BIGINT,
  currency VARCHAR(32),
  created TIMESTAMP,
  value_date TIMESTAMP,
  account_id VARCHAR(32),
  transaction_id VARCHAR(32),
  scheme VARCHAR(32),
  PRIMARY KEY (account_id, posting_group_id),
  INDEX (transaction_id),
  INDEX (posting_group_id)
);
TRUNCATE TABLE ledger_accounts;
`
)

// Config is for ledgerClient
type Config struct {
	NumAccounts int           `toml:"num_accounts"`
	Interval    time.Duration `toml:"interval"`
	Concurrency int           `toml:"concurrency"`
	TxnMode     string        `toml:"txn_mode"`
}

// ClientCreator creates ledgerClient
type ClientCreator struct {
	Cfg *Config
}

// Create ...
func (l ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &ledgerClient{
		Config: l.Cfg,
	}
}

// ledgerClient simulates a complete record of financial transactions over the
// life of a bank (or other company).
type ledgerClient struct {
	*Config
	wg   sync.WaitGroup
	stop int32
	db   *sql.DB
}

func (c *ledgerClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	var err error
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/test?multiStatements=true", node.IP, node.Port)

	log.Infof("start to init...")
	db, err := util.OpenDB(dsn, 1)
	if err != nil {
		log.Fatalf("[ledgerClient] create db client error %v", err)
	}
	_, err = db.Exec(fmt.Sprintf("set @@global.tidb_txn_mode = '%s';", c.TxnMode))
	if err != nil {
		log.Fatalf("[ledgerClient] set txn_mode failed: %v", err)
	}
	time.Sleep(5 * time.Second)
	c.db, err = util.OpenDB(dsn, c.Concurrency)
	if err != nil {
		return err
	}
	defer func() {
		log.Infof("init end...")
	}()

	if _, err := c.db.Exec(stmtDrop); err != nil {
		log.Fatalf("execute statement %s error %v", stmtDrop, err)
	}

	if _, err := c.db.Exec(stmtCreate); err != nil {
		log.Fatalf("execute statement %s error %v", stmtCreate, err)
	}
	var wg sync.WaitGroup
	type Job struct {
		begin, end int
	}
	ch := make(chan Job)
	for i := 0; i < insertConcurrency; i++ {
		wg.Add(1)
		start := time.Now()
		go func() {
			defer wg.Done()
			for job := range ch {
				select {
				case <-ctx.Done():
					return
				default:
				}
				args := make([]string, 0, insertBatchSize)
				for i := job.begin; i < job.end; i++ {
					args = append(args, fmt.Sprintf(`(%v, 0, "acc%v", 0, 0)`, rand.Int63(), i))
				}

				query := fmt.Sprintf(`INSERT INTO ledger_accounts (posting_group_id, amount,account_id, causality_id, balance) VALUES %s`, strings.Join(args, ","))
				err := util.RunWithRetry(ctx, retryLimit, 3*time.Second, func() error {
					_, err := c.db.Exec(query)
					if util.IsErrDupEntry(err) {
						return nil
					}
					return err
				})

				if err != nil {
					log.Fatalf("exec %s err %s", query, err)
				}
				log.Infof("insert %d accounts, takes %s", job.end-job.begin, time.Since(start))
			}
		}()
	}

	var begin, end int
	for begin = 1; begin <= c.NumAccounts; begin = end {
		end = begin + insertBatchSize
		if end > c.NumAccounts {
			end = c.NumAccounts + 1
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
		log.Warnf("ledgerClient initialize is cancel")
		return nil
	default:
	}

	c.startVerify(ctx, c.db)
	return nil
}

func (c *ledgerClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (c *ledgerClient) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	log.Infof("start to test...")
	defer func() {
		log.Infof("test end...")
	}()
	var wg sync.WaitGroup
	for i := 0; i < c.Concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if err := c.ExecuteLedger(c.db); err != nil {
					log.Errorf("exec failed %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
	return nil
}

type postingRequest struct {
	Group               int64
	AccountA, AccountB  string
	Amount              int    // deposited on AccountA, removed from AccountB
	Transaction, Scheme string // opaque
}

// ExecuteLedger is run case
func (c *ledgerClient) ExecuteLedger(db *sql.DB) error {
	if atomic.LoadInt32(&c.stop) != 0 {
		return errors.New("ledgerClient stopped")
	}

	c.wg.Add(1)
	defer c.wg.Done()

	req := postingRequest{
		Group:    rand.Int63(),
		AccountA: fmt.Sprintf("acc%d", rand.Intn(c.NumAccounts)+1),
		AccountB: fmt.Sprintf("acc%d", rand.Intn(c.NumAccounts)+1),
		Amount:   rand.Intn(maxTransfer),
	}

	if req.AccountA == req.AccountB {
		// The code we use throws a unique constraint violation since we
		// try to insert two conflicting primary keys. This isn't the
		// interesting case.
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	defer tx.Rollback()

	if err = c.doPosting(tx, req); err != nil {
		log.Errorf("doPosting error: %v", err)
		return errors.Trace(err)
	}

	return nil
}

func getLast(tx *sql.Tx, accountID string) (lastCID int64, lastBalance int64, err error) {
	err = tx.QueryRow(fmt.Sprintf(`SELECT causality_id, balance FROM ledger_accounts
		WHERE account_id = "%v" ORDER BY causality_id DESC LIMIT 1 FOR UPDATE`, accountID)).
		Scan(&lastCID, &lastBalance)
	return
}

func (c *ledgerClient) doPosting(tx *sql.Tx, req postingRequest) error {
	//start := time.Now()
	var cidA, balA, cidB, balB int64
	var err error
	cidA, balA, err = getLast(tx, req.AccountA)
	if err != nil {
		return err
	}
	cidB, balB, err = getLast(tx, req.AccountB)
	if err != nil {
		return err
	}
	query := fmt.Sprintf(`
INSERT INTO ledger_accounts (
  posting_group_id,
  amount,
  account_id,
  causality_id,
  balance
)
VALUES (
  %[1]v,	-- posting_group_id
  %[2]v, 	-- amount
  "%[3]v", 	-- account_id (A)
  %[4]v, 	-- causality_id
  %[5]v+%[2]v	-- (new) balance
), (
  %[1]v,   -- posting_group_id
 -%[2]v,   -- amount
  "%[6]v", -- account_id (B)
  %[7]v,   -- causality_id
  %[8]v-%[2]v -- (new) balance
)`, req.Group, req.Amount,
		req.AccountA, cidA+1, balA,
		req.AccountB, cidB+1, balB)
	if _, err := tx.Exec(query); err != nil {
		return errors.Trace(err)
	}
	if err := tx.Commit(); err != nil {
		return errors.Trace(err)
	}

	//ledgerTxnDuration.Observe(time.Since(start).Seconds())
	return nil
}

func (c *ledgerClient) startVerify(ctx context.Context, db *sql.DB) {
	c.verify(db)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.Interval):
				c.verify(db)
			}
		}
	}()
}

func (c *ledgerClient) verify(db *sql.DB) error {
	start := time.Now()
	tx, err := db.Begin()
	if err != nil {
		//ledgerVerifyFailedCounter.Inc()
		return errors.Trace(err)
	}
	defer tx.Rollback()

	var total int64
	err = tx.QueryRow(`
select sum(balance) total from
  (select account_id, max(causality_id) max_causality_id from ledger_accounts group by account_id) last
  join ledger_accounts
    on last.account_id = ledger_accounts.account_id and last.max_causality_id = ledger_accounts.causality_id`).Scan(&total)
	if err != nil {
		//ledgerVerifyFailedCounter.Inc()
		return errors.Trace(err)
	}

	//ledgerVerifyDuration.Observe(time.Since(start).Seconds())
	if total != 0 {
		log.Errorf("check total balance got %v", total)
		atomic.StoreInt32(&c.stop, 1)
		c.wg.Wait()
		log.Fatalf("check total balance got %v", total)
	}

	log.Infof("verify ok, cost time: %v", time.Since(start))
	return nil
}
