// Copyright 2020 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package backup

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

const (
	initialBalance  = 1000
	maxTransfer     = 100
	systemAccountID = 0
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var stmtsCreate = []string{
	`CREATE TABLE IF NOT EXISTS accounts (
		id INT,
		balance INT NOT NULL,
		name VARCHAR(32),
		remark VARCHAR(2048),
		PRIMARY KEY (id),
		UNIQUE INDEX byName (name)
	);`,
	`CREATE TABLE IF NOT EXISTS transaction (
		id INT,
		booking_date TIMESTAMP DEFAULT NOW(),
		txn_date TIMESTAMP DEFAULT NOW(),
		txn_ref VARCHAR(32),
		remark VARCHAR(2048),
		PRIMARY KEY (id),
		UNIQUE INDEX byTxnRef (txn_ref)
	);`,
	`CREATE TABLE IF NOT EXISTS transaction_leg (
		id INT AUTO_INCREMENT,
		account_id INT,
		amount INT NOT NULL,
		running_balance INT NOT NULL,
		txn_id INT,
		remark VARCHAR(2048),
		PRIMARY KEY (id)
	);`,
	`TRUNCATE TABLE accounts;`,
	`TRUNCATE TABLE transaction;`,
	`TRUNCATE TABLE transaction_leg;`,
}

// Features means the feature on TiDB we can turn on and off
type Features struct {
	Pessimistic bool
	ReplicaRead string
	AsyncCommit bool
	OnePC       bool
}

// Config means the config of this test case
type Config struct {
	NumAccounts int
	Concurrency int
	Contention  string
	// run backup once every BackupInterval
	BackupInterval time.Duration
	// run restore once every RestoreInterval
	RestoreInterval time.Duration
	DbName          string
	RetryLimit      int
	// will backup to BackupURI/full-$nextBackupIndex
	BackupURI url.URL
}

type backupClient struct {
	features         Features
	config           Config
	db               *sql.DB
	txnID            int32
	lastBackupTs     uint64
	nextRestoreIndex int
	nextBackupIndex  int
}

func (c *backupClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}
	var err error
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/%s", node.IP, node.Port, c.config.DbName)
	log.Infof("[%s] start to init...", c)
	c.db, err = util.OpenDB(dsn, c.config.Concurrency)
	if err != nil {
		return err
	}
	defer func() {
		log.Infof("[%s] init end...", c)
	}()
	c.applyConfig()
	c.db, err = util.OpenDB(dsn, c.config.Concurrency)
	c.db.SetMaxOpenConns(100)
	if err != nil {
		return err
	}
	c.createTables()
	c.initData(ctx)
	return nil
}

// Refused Bequest, just for implement Client interface
func (c *backupClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

// Start the test
func (c *backupClient) Start(ctx context.Context, _ interface{}, _ []cluster.ClientNode) error {
	log.Infof("[%s] start to test...", c)
	var restoringLock sync.RWMutex
	c.startTransactions(&restoringLock)
	go c.startBackup(&restoringLock)
	go c.startRestore(&restoringLock)
	<-ctx.Done()
	return nil
}

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func (c *backupClient) applyConfig() {
	var err error
	stmt := fmt.Sprintf("set @@tidb_replica_read = '%s'", c.features.ReplicaRead)
	if _, err = c.db.Exec(stmt); err != nil {
		log.Errorf("[%s] tidb_replica_read set fail: %v", c, err)
	}
	if c.features.AsyncCommit {
		_, err = c.db.Exec("set @@global.tidb_enable_async_commit = 1;")
	} else {
		_, err = c.db.Exec("set @@global.tidb_enable_async_commit = 0;")
	}
	if err != nil {
		log.Fatalf("[%s] set async commit failed: %v", c, err)
	}
	if c.features.OnePC {
		_, err = c.db.Exec("set @@global.tidb_enable_1pc = 1;")
	} else {
		_, err = c.db.Exec("set @@global.tidb_enable_1pc = 0;")
	}
	if err != nil {
		log.Fatalf("[%s] set 1PC failed: %v", c, err)
	}
	if c.features.Pessimistic {
		_, err = c.db.Exec("set @@global.tidb_txn_mode = 'pessimistic';")
	} else {
		_, err = c.db.Exec("set @@global.tidb_txn_mode = 'optimistic';")
	}
	if err != nil {
		log.Fatalf("[%s] set txn_mode failed: %v", c, err)
	}
	time.Sleep(5 * time.Second)
}

func (c *backupClient) createTables() {
	for _, stmt := range stmtsCreate {
		if _, err := c.db.Exec(stmt); err != nil {
			log.Fatalf("[%s] execute statement %s error %v", c, stmt, err)
		}
	}
}

func (c *backupClient) initData(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < c.config.NumAccounts; i++ {
		stmt := fmt.Sprintf(`INSERT IGNORE INTO accounts (id, balance, name, remark) VALUES (%d, %d, "account %d", "%s");`, i, initialBalance, i, randomString(36))
		wg.Add(1)
		go func(db *sql.DB) {
			defer wg.Done()
			err := util.RunWithRetry(ctx, c.config.RetryLimit, 5*time.Second, func() error {
				_, err := db.Exec(stmt)
				if util.IsErrDupEntry(err) {
					return nil
				}
				return err
			})
			if err != nil {
				log.Fatalf("[%s] exec %s err %v", c, stmt, err)
			}
		}(c.db)
	}
	wg.Wait()
}

func (c *backupClient) backup() error {
	log.Infof("[%s] Try backup once", c)
	queryString := fmt.Sprintf(`BACKUP DATABASE * TO '%s' LAST_BACKUP = %d;`, c.backupURI(c.nextBackupIndex), c.lastBackupTs)
	row := c.db.QueryRow(queryString)
	var ignore string
	var lastBackupTs uint64
	err := row.Scan(&ignore, &ignore, &lastBackupTs, &ignore, &ignore)
	if err != nil {
		log.Warnf("[%s] Backup failed, err: %v", c, err)
		return err
	}
	log.Infof("[%s] Back up %d success, this increment include updates from %d to %d", c, c.nextBackupIndex, c.lastBackupTs, lastBackupTs)
	c.lastBackupTs = lastBackupTs
	c.nextBackupIndex++
	return nil
}

func (c *backupClient) backupURI(index int) []byte {
	backupURI := c.config.BackupURI
	backupURI.Path += fmt.Sprintf("/full-%d", index)
	backupURIString, _ := backupURI.MarshalBinary()
	return backupURIString
}

func (c *backupClient) restore() {
	log.Infof("[%s] Start restore...", c)
	for ; c.nextRestoreIndex < c.nextBackupIndex; c.nextRestoreIndex++ {
		backupURI := c.backupURI(c.nextRestoreIndex)
		log.Infof("[%s] Restoring from %s ...", c, backupURI)
		_, err := c.db.Exec(fmt.Sprintf(`RESTORE DATABASE * FROM '%s'`, backupURI))
		if err != nil {
			// no error should occur during restore
			log.Fatalf("[%s] Failed, err: %v", c, err)
		} else {
			log.Infof("[%s] Success", c)
		}
	}
	c.lastBackupTs = 0
	log.Infof("[%s] Restore success", c)
}

func (c *backupClient) transferOnce() error {
	from, to := rand.Intn(c.config.NumAccounts), rand.Intn(c.config.NumAccounts)
	if c.config.Contention == "high" {
		// Use the first account number we generated as a coin flip to
		// determine whether we're transferring money into or out of
		// the system account.
		if from > c.config.NumAccounts/2 {
			from = systemAccountID
		} else {
			to = systemAccountID
		}
	}
	if from == to {
		return nil
	}
	amount := rand.Intn(maxTransfer)

	tx, err := c.db.Begin()
	if err != nil {
		return errors.Trace(err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	rows, err := tx.Query(fmt.Sprintf("SELECT id, balance FROM accounts WHERE id IN (%d, %d) FOR UPDATE", from, to))
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

	insertTxn := `INSERT INTO transaction (id, txn_ref, remark) VALUES (?, ?, ?)`
	insertTxnLeg := `INSERT INTO transaction_leg (account_id, amount, running_balance, txn_id, remark) VALUES (?, ?, ?, ?, ?)`
	updateAcct := `UPDATE accounts SET balance = ? WHERE id = ?`
	txnID := atomic.AddInt32(&c.txnID, 1)
	if _, err := tx.Exec(insertTxn, txnID, fmt.Sprintf("txn %d", txnID), randomString(36)); err != nil {
		_ = tx.Rollback()
		return errors.Trace(err)
	}
	if _, err = tx.Exec(insertTxnLeg, from, -amount, fromBalance-amount, txnID, randomString(36)); err != nil {
		_ = tx.Rollback()
		return errors.Trace(err)
	}
	if _, err = tx.Exec(insertTxnLeg, to, amount, toBalance+amount, txnID, randomString(36)); err != nil {
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

	return tx.Commit()
}

func (c *backupClient) startRestore(restoringLock *sync.RWMutex) {
	for {
		time.Sleep(c.config.RestoreInterval)
		// according to the document, no other operations are allowed to access the database when restoring
		log.Infof("[%s] Try to restore once, waiting for other task finish...", c)
		restoringLock.Lock()
		log.Infof("[%s] Other task finished", c)
		// now no other workers are operating the database, let's do the check work
		// first backup once, so we should build the current state of this database with all backups
		log.Infof("[%s] Backup once before restore", c)
		err := util.RunWithRetry(context.Background(), c.config.RetryLimit, 5*time.Second, func() error {
			err := c.backup()
			return err
		})
		if err != nil {
			log.Fatalf("[%s] failed to backup after try for %d times before restore, err: %v", c, c.config.RetryLimit, err)
		}
		oldNextRestoreIndex := c.nextRestoreIndex
		// and then do the saveState, clearDB, restore and check work
		balances := c.saveState()
		c.clearDB()
		c.restore()
		c.checkRestoreSuccess(balances)
		log.Infof("[%s] Restore from backup %d-%d pass the validate", c, oldNextRestoreIndex, c.nextRestoreIndex-1)
		// the old backup files are invalidated after restore
		// so backup once after restore
		log.Infof("[%s] Backup once after restore", c)
		err = util.RunWithRetry(context.Background(), c.config.RetryLimit, 5*time.Second, func() error {
			err := c.backup()
			return err
		})
		if err != nil {
			log.Fatalf("[%s] failed to backup after try for %d times after restore, err: %v", c, c.config.RetryLimit, err)
		}
		restoringLock.Unlock()
	}
}

func (c *backupClient) startBackup(restoringLock *sync.RWMutex) {
	for {
		time.Sleep(c.config.BackupInterval)
		// prevent restore when there is a living backup work
		restoringLock.RLock()
		err := util.RunWithRetry(context.Background(), c.config.RetryLimit, 5*time.Second, func() error {
			err := c.backup()
			return err
		})
		if err != nil {
			log.Fatalf("[%s] failed to backup after try for %d times, err: %v", c, c.config.RetryLimit, err)
		}
		restoringLock.RUnlock()
	}
}

func (c *backupClient) startTransactions(restoringLock *sync.RWMutex) {
	for i := 0; i < c.config.Concurrency; i++ {
		go func() {
			for {
				// prevent restore when there is a living transfer
				restoringLock.RLock()
				if err := c.transferOnce(); err != nil {
					log.Errorf("[%s] move money err %v", c, err)
					return
				}
				restoringLock.RUnlock()
			}
		}()
	}
}

func (c *backupClient) logBalances(path string, balances []uint64) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, balance := range balances {
		_, err := f.WriteString(fmt.Sprintf("%d\n", balance))
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *backupClient) checkRestoreSuccess(expectedBalances []uint64) {
	// query the restored result and check whether it matched with the origin result
	// if incremental backup works as expected, the result should be just equal
	rows, err := c.db.Query(`SELECT balance FROM accounts ORDER BY id;`)
	if err != nil {
		log.Fatal(err)
	}
	var balances []uint64
	for rows.Next() {
		var balance uint64
		if err := rows.Scan(&balance); err != nil {
			log.Fatal(err)
		}
		balances = append(balances, balance)
	}
	matched := len(expectedBalances) == len(balances)
	for i, balance := range expectedBalances {
		if !matched {
			break
		}
		if balance != balances[i] {
			matched = false
		}
	}
	if !matched {
		if err = c.logBalances("./before-recover.log", expectedBalances); err != nil {
			log.Fatal(err)
		}
		if err = c.logBalances("./after-recover.log", balances); err != nil {
			log.Fatal(err)
		}
		log.Fatalf("[%s] Balance not match after recover! Check before-recover.log and after-recover.log for the difference", c)
	}
}

func (c *backupClient) clearDB() {
	log.Infof("[%s] Clear the database before restore...", c)
	// then drop the tables, I did not find a better way to clear the storage
	err := util.RunWithRetry(context.Background(), c.config.RetryLimit, 5*time.Second, func() error {
		_, err := c.db.Exec(`drop table accounts;`)
		return err
	})
	if err != nil {
		log.Fatalf("[%s] drop table err %v", c, err)
	}
	err = util.RunWithRetry(context.Background(), c.config.RetryLimit, 5*time.Second, func() error {
		_, err := c.db.Exec(`drop table transaction;`)
		return err
	})
	if err != nil {
		log.Fatalf("[%s] drop table err %v", c, err)
	}
	err = util.RunWithRetry(context.Background(), c.config.RetryLimit, 5*time.Second, func() error {
		_, err := c.db.Exec(`drop table transaction_leg;`)
		return err
	})
	if err != nil {
		log.Fatalf("[%s] drop table err %v", c, err)
	}
	log.Infof("[%s] Database clean now", c)
}

func (c *backupClient) saveState() []uint64 {
	// currently we just check all balances
	// todo: check transaction and transaction_leg, though these tables might be large we can check all fields' checksum
	var balances []uint64
	rows, err := c.db.Query(`SELECT balance FROM accounts ORDER BY id;`)
	if err != nil {
		log.Fatal(err)
	}
	var balance uint64
	for rows.Next() {
		if err := rows.Scan(&balance); err != nil {
			log.Fatal(err)
		}
		balances = append(balances, balance)
	}
	return balances
}

func (c *backupClient) String() string {
	return "backup"
}

// ClientCreator ...
type ClientCreator struct {
	Cfg      Config
	Features Features
}

// Create a Client
func (c ClientCreator) Create(_ cluster.ClientNode) core.Client {
	return &backupClient{
		features: c.Features,
		config:   c.Cfg,
	}
}
