package deadlock

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"
	"github.com/pingcap/tipocket/util"
)

type deadlockCase struct {
	dsn      string
	interval time.Duration
	timeout  time.Duration
	db       *sql.DB
}

func newDeadlockCase(dsn string, interval time.Duration, duration time.Duration) *deadlockCase {
	return &deadlockCase{
		dsn:      dsn,
		interval: interval,
		timeout:  duration,
	}
}

func (d *deadlockCase) initialize(ctx context.Context) error {
	log.Info("[deadlock] Initialize database")

	var err error
	// Connect to DB
	d.db, err = util.OpenDB(d.dsn, 2)
	if err != nil {
		return errors.Wrap(err, "Open db failed")
	}

	if err = d.checkCluster(); err != nil {
		return err
	}

	// Initialize DB
	util.MustExec(d.db, "DROP TABLE IF EXISTS deadlock")
	util.MustExec(d.db, "CREATE TABLE deadlock (v int)")
	util.MustExec(d.db, "INSERT INTO deadlock VALUES (0), (1)")
	return nil
}

func (d *deadlockCase) execute(ctx context.Context) error {
	ticker := time.NewTicker(d.interval)
	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			d.deadlock()
		}
	}
}

func (d *deadlockCase) checkCluster() error {
	var config string
	err := d.db.QueryRow("SELECT VARIABLE_VALUE FROM `INFORMATION_SCHEMA`.`SESSION_VARIABLES` WHERE `VARIABLE_NAME` = 'tidb_config'").Scan(&config)
	if err != nil {
		return errors.Annotate(err, "resolve tidb config")
	}
	var cfg struct {
		PessimisticTxn struct {
			Enable bool `json:"enable"`
		} `json:"pessimistic-txn"`
	}
	if err = json.Unmarshal([]byte(config), &cfg); err != nil {
		return errors.Annotate(err, "parse tidb config")
	}
	if !cfg.PessimisticTxn.Enable {
		return errors.New("pessimistic txn is disabled")
	}
	return nil
}

func (d *deadlockCase) deadlock() {
	d.doWithConns(2, func(ctx context.Context, conns []*sql.Conn) error {

		exec := func(i int, sql string, optAnn ...string) error {
			if _, err := conns[i].ExecContext(ctx, sql); err != nil {
				ann := "[deadlock] conns[" + strconv.Itoa(i) + "].exec(" + sql + ")"
				if len(optAnn) > 0 {
					ann = optAnn[0]
				}
				return errors.Annotate(err, ann)
			}
			return nil
		}

		mustExec := func(i int, sql string, optAnn ...string) {
			if err := exec(i, sql, optAnn...); err != nil {
				log.Fatalf(err.Error())
			}
		}

		mustExec(0, "BEGIN PESSIMISTIC")
		mustExec(1, "BEGIN PESSIMISTIC")

		defer func() {
			mustExec(0, "ROLLBACK")
			mustExec(1, "ROLLBACK")
		}()

		mustExec(0, "UPDATE deadlock SET v = v + 1 WHERE v = 0")
		mustExec(1, "UPDATE deadlock SET v = v + 1 WHERE v = 1")

		errs := make(chan error, 2)
		go func() { errs <- exec(0, "UPDATE deadlock SET v = v + 1 WHERE v = 1") }()
		go func() { errs <- exec(1, "UPDATE deadlock SET v = v + 1 WHERE v = 0") }()

		select {
		case <-time.After(d.timeout):
			log.Fatalf("[deadlock] Detect deadlock timeout: %s", d.timeout.String())

		case err1 := <-errs:
			err2 := <-errs
			if err1 == nil {
				err1, err2 = err2, err1
			}
			if err1 == nil || err2 != nil || !isDeadlockError(err1) {
				log.Fatalf("[deadlock] Expect err1 is deadlock error, got: \"%v\"; err2 is nil, got: \"%v\"", err1, err2)
			}
		}

		return nil
	})
}

func (d *deadlockCase) doWithConns(n int, action func(ctx context.Context, conns []*sql.Conn) error) error {
	ctx := context.Background()
	conns := make([]*sql.Conn, n)
	defer func() {
		for _, conn := range conns {
			if conn != nil {
				conn.Close()
			}
		}
	}()
	for i := range conns {
		conn, err := d.db.Conn(ctx)
		if err != nil {
			return err
		}
		conns[i] = conn
	}
	return action(ctx, conns)
}

func isDeadlockError(err error) bool {
	return strings.Contains(err.Error(), "Deadlock")
}
