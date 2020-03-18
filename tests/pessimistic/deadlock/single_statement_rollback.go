package deadlock

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ngaut/log"
	"github.com/pingcap/errors"

	"github.com/pingcap/tipocket/util"
)

type singleStatementRollbackCase struct {
	dsn      string
	tableNum int
	interval time.Duration
	db       *sql.DB
	query    string
	chs      []chan struct{}
}

func newSingleStatementRollbackCase(dsn string, tableNum int, interval time.Duration) *singleStatementRollbackCase {
	return &singleStatementRollbackCase{
		dsn:      dsn,
		tableNum: tableNum,
		interval: interval,
	}
}

func (s *singleStatementRollbackCase) initialize(ctx context.Context) error {
	log.Info("[single statement rollback] Initialize database")
	if err := s.check(); err != nil {
		return err
	}
	if err := s.openDB(); err != nil {
		return err
	}
	if err := s.createTables(); err != nil {
		return err
	}
	return nil
}

func (s *singleStatementRollbackCase) check() error {
	if s.tableNum <= 1 {
		return errors.New("tableNum should be greater than 1")
	}
	// TODO(yeya24): add that check once we can get tikv config.
	// This case should check 'split-region-on-table'.
	// However we can't get tikv config through http api in release-3.0.
	return nil
}

func (s *singleStatementRollbackCase) openDB() error {
	// Connect to DB
	log.Infof("[deadlock] DSN: %s", s.dsn)
	var err error
	s.db, err = util.OpenDB(s.dsn, 2)
	if err != nil {
		return errors.Wrap(err, "Open db failed")
	}
	return nil
}

func (s *singleStatementRollbackCase) createTables() error {
	for i := 0; i < s.tableNum; i++ {
		util.MustExec(s.db, fmt.Sprintf("DROP TABLE IF EXISTS ssr%d", i))
		util.MustExec(s.db,
			fmt.Sprintf("CREATE TABLE ssr%d(id INT PRIMARY KEY, v BIGINT)", i))
		util.MustExec(s.db, fmt.Sprintf("INSERT INTO ssr%d VALUES (0, 0)", i))
	}

	// UPDATE ssr0, ssr1, .. SET ssr0.v = ssr0.v + 1, ssr1.v = ssr1.v + 1, .. where ssr0.id = 0, ssr1.id = 0, ..
	b := bytes.NewBufferString("UPDATE ")
	for i := 0; i < s.tableNum; i++ {
		b.WriteString(fmt.Sprintf("ssr%d", i))
		if i != s.tableNum-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(" SET ")
	for i := 0; i < s.tableNum; i++ {
		b.WriteString(fmt.Sprintf("ssr%d.v = ssr%d.v + 1", i, i))
		if i != s.tableNum-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(" WHERE ")
	for i := 0; i < s.tableNum; i++ {
		b.WriteString(fmt.Sprintf("ssr%d.id = 0", i))
		if i != s.tableNum-1 {
			b.WriteString(" or ")
		}
	}
	s.query = b.String()
	return nil
}

func (s *singleStatementRollbackCase) execute(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-ctx.Done():
			for _, ch := range s.chs {
				close(ch)
			}
			return nil

		case <-ticker.C:
			s.singleStatementRollback()
		}
	}
}

func (s *singleStatementRollbackCase) singleStatementRollback() {
	if len(s.chs) == 0 {
		for i := 0; i < s.tableNum; i++ {
			ch := make(chan struct{})
			go func() {
				for {
					_, ok := <-ch
					if !ok {
						return
					}
					mustExec := func(ctx context.Context, conn *sql.Conn, query string) {
						if _, err := conn.ExecContext(ctx, query); err != nil {
							log.Fatalf("[single statement rollback] Execute \"%v\" failed: %v", query, err)
						}
					}
					ctx := context.Background()
					conn, err := s.db.Conn(ctx)
					if err != nil {
						log.Fatalf("[single statement rollback] Create connection failed: %v", err)
					}
					mustExec(ctx, conn, "BEGIN PESSIMISTIC")
					mustExec(ctx, conn, s.query)
					mustExec(ctx, conn, "COMMIT")
					conn.Close()
				}
			}()
			s.chs = append(s.chs, ch)
		}
	}
	for _, ch := range s.chs {
		ch <- struct{}{}
	}
}
