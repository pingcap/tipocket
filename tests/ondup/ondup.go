package ondup

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	tmysql "github.com/pingcap/parser/mysql"
	"github.com/pingcap/parser/terror"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

const stmtCreate = `
CREATE TABLE IF NOT EXISTS on_dup (
  id INT,
  name VARCHAR(64),
  score INT,
  PRIMARY KEY (id),
  UNIQUE INDEX byName (name)
)`

// Config is for ondupClient
type Config struct {
	DBName     string `toml:"db_name"`
	NumRows    int    `toml:"num_rows"`
	RetryLimit int    `toml:"retry_limit"`
}

// ondupClient tests insert statement with on duplicate update.
// We start insert worker, replace worker and delete worker to
// run concurrently, each worker uses a random id to execute the
// statement at a time, so there will be retries, but should never fail.
type ondupClient struct {
	*Config
	db    *sql.DB
	errCh chan error
}

// ClientCreator creates ondupClient
type ClientCreator struct {
	Cfg *Config
}

// Create ...
func (l ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &ondupClient{
		Config: l.Cfg,
		errCh:  make(chan error, 3),
	}
}

func (c *ondupClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	var err error
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/%s", node.IP, node.Port, c.DBName)

	log.Infof("start to init...")
	c.db, err = util.OpenDB(dsn, 20)
	if err != nil {
		return fmt.Errorf("[on_dup] create db client error %v", err)
	}
	if _, err := c.db.Exec(stmtCreate); err != nil {
		return errors.Trace(err)
	}
	_, err = c.db.Exec(`TRUNCATE TABLE on_dup`)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *ondupClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (c *ondupClient) Invoke(ctx context.Context, node cluster.ClientNode, r interface{}) core.UnknownResponse {
	panic("implement me")
}

func (c *ondupClient) NextRequest() interface{} {
	panic("implement me")
}

func (c *ondupClient) DumpState(ctx context.Context) (interface{}, error) {
	panic("implement me")
}

func (c *ondupClient) Start(ctx context.Context, cfg interface{}, clientNode []cluster.ClientNode) error {
	childCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	go c.runInsert(childCtx)
	go c.runDelete(childCtx)
	go c.runReplace(childCtx)
	select {
	case err := <-c.errCh:
		return errors.Trace(err)
	case <-ctx.Done():
		return nil
	}
}

func (c *ondupClient) runInsert(ctx context.Context) {
	ran := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	for {
		id := ran.Intn(c.NumRows)
		name := strconv.Itoa(int(ran.Intn(c.NumRows)))
		score := ran.Intn(c.NumRows)
		query := fmt.Sprintf("INSERT on_dup VALUES (%d, '%s', %d) ON DUPLICATE KEY UPDATE score = values(score)", id, name, score)
		err := util.RunWithRetry(ctx, c.RetryLimit, 3*time.Second, func() error {
			_, err := c.db.Exec(query)
			if err != nil {
				log.Errorf("Exec [query:%s] failed: %v, retry", query, err)
			}
			return err
		})
		if err != nil {
			ignoreErrs := []terror.ErrCode{
				tmysql.ErrUnknown,
				tmysql.ErrDupEntry,
			}
			if util.IgnoreErrors(err, ignoreErrs) {
				log.Warnf("[on_dup] skip error: %v", err)
				continue
			}
			log.Errorf("Exec [query:%s] failed: %v, send error to channel", query, err)
			c.errCh <- errors.Trace(err)
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (c *ondupClient) runDelete(ctx context.Context) {
	ran := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	for {
		id := ran.Intn(c.NumRows)
		query := fmt.Sprintf("DELETE from on_dup where id = %d", id)
		err := util.RunWithRetry(ctx, c.RetryLimit, 3*time.Second, func() error {
			_, err := c.db.Exec(query)
			if err != nil {
				log.Errorf("Exec [query:%s] failed: %v, retry", query, err)
			}
			return err
		})
		if err != nil {
			ignoreErrs := []terror.ErrCode{
				tmysql.ErrUnknown,
				tmysql.ErrKeyNotFound,
			}
			if util.IgnoreErrors(err, ignoreErrs) {
				log.Warnf("[on_dup] skip error: %v", err)
				continue
			}
			log.Errorf("Exec [query:%s] failed: %v, send error to channel", query, err)
			c.errCh <- errors.Trace(err)
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (c *ondupClient) runReplace(ctx context.Context) {
	ran := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	for {
		id := ran.Intn(c.NumRows)
		name := strconv.Itoa(int(ran.Intn(c.NumRows)))
		score := ran.Intn(c.NumRows)
		query := fmt.Sprintf("REPLACE on_dup VALUES (%d, '%s', %d)", id, name, score)
		err := util.RunWithRetry(ctx, c.RetryLimit, 3*time.Second, func() error {
			_, err := c.db.Exec(query)
			if err != nil {
				log.Errorf("Exec [query:%s] failed: %v, retry", query, err)
			}
			return err
		})
		if err != nil {
			ignoreErrs := []terror.ErrCode{
				tmysql.ErrUnknown,
			}
			if util.IgnoreErrors(err, ignoreErrs) {
				log.Warnf("[on_dup] skip error: %v", err)
				continue
			}
			log.Errorf("Exec [query:%s] failed: %v, send error to channel", query, err)
			c.errCh <- errors.Trace(err)
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
