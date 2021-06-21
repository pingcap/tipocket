package crud

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/util"
)

var errNotExist = errors.New("the request row does not exist")

var tableNames = []string{
	"crud_users",
	"crud_posts",
}

// Config is for crudClient
type Config struct {
	DBName              string
	UserCount           int
	PostCount           int
	UpdateUsers         int
	UpdatePosts         int
	Interval            time.Duration
	RetryLimit          int
	TxnMode             string
	TiFlashDataReplicas int
}

// ClientCreator creates crudClient
type ClientCreator struct {
	Cfg *Config
}

// Create ...
func (l ClientCreator) Create(node cluster.ClientNode) core.Client {
	return &crudClient{
		Config:  l.Cfg,
		userIDs: newIDList(),
		postIDs: newIDList(),
		rnd:     rand.New(rand.NewSource(time.Now().Unix())),
	}
}

type crudClient struct {
	*Config
	sync.Mutex
	userIDs *idList
	postIDs *idList
	rnd     *rand.Rand

	db *sql.DB
}

func (c *crudClient) String() string {
	return "crud"
}

func (c *crudClient) SetUp(ctx context.Context, _ []cluster.Node, clientNodes []cluster.ClientNode, idx int) error {
	if idx != 0 {
		return nil
	}

	var err error
	node := clientNodes[idx]
	dsn := fmt.Sprintf("root@tcp(%s:%d)/%s", node.IP, node.Port, c.DBName)

	log.Infof("start to init...")
	defer func() {
		log.Infof("init end...")
	}()
	c.db, err = util.OpenDB(dsn, 1)
	if err != nil {
		log.Fatalf("[%s] create db client error %v", c, err)
	}
	if _, err = c.db.Exec(fmt.Sprintf("set @@global.tidb_txn_mode = '%s';", c.TxnMode)); err != nil {
		log.Fatalf("[%s] set txn_mode failed: %v", c, err)
	}

	time.Sleep(5 * time.Second)
	util.MustExec(c.db, "DROP TABLE IF EXISTS crud_users, crud_posts")
	util.MustExec(c.db, "CREATE TABLE crud_users (id BIGINT PRIMARY KEY, name VARCHAR(16), posts BIGINT)")
	util.MustExec(c.db, "CREATE TABLE crud_posts (id BIGINT PRIMARY KEY, author BIGINT, title VARCHAR(128))")
	if c.TiFlashDataReplicas > 0 {
		// create tiflash replica
		maxSecondsBeforeTiFlashAvail := 1000
		for _, tableName := range tableNames {
			if err := util.SetAndWaitTiFlashReplica(ctx, c.db, c.DBName, tableName, c.TiFlashDataReplicas, maxSecondsBeforeTiFlashAvail); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *crudClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

func (c *crudClient) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	log.Infof("[%s] start to test...", c)
	defer func() {
		log.Infof("[%s] test end...", c)
	}()
	if err := c.ExecuteCrud(c.db); err != nil {
		log.Errorf("[%s] execute failed %v", c, err)
		return err
	}

	return nil
}

// ExecuteCrud runs a single-thread test because crudClient does not support multi-threading.
func (c *crudClient) ExecuteCrud(db *sql.DB) error {
	var newUsers, deleteUsers, newPosts, newAuthors, deletePosts []int64
	defer func() {
		log.Infof("[%s] newUsers %v, deleteUsers %v, newPosts %v, newAuthors %v, deletePosts %v", c, newUsers, deleteUsers, newPosts, newAuthors, deletePosts)
	}()

	// Add new users.
	for i := 0; i < c.UpdateUsers && c.userIDs.len() < c.UserCount+c.UpdateUsers; i++ {
		id := c.userIDs.allocID()
		log.Infof("[%s] create user %d", c, id)
		if err := c.createUser(db, id); err != nil {
			return errors.Trace(err)
		}
		newUsers = append(newUsers, id)
	}

	// Delete random users.
	for i := 0; i < c.UpdateUsers && c.userIDs.len() > c.UserCount; i++ {
		id, ok := c.userIDs.randomID()
		if !ok {
			break
		}
		log.Infof("[%s] delete user %d", c, id)
		if err := c.deleteUser(id); err != nil {
			return errors.Trace(err)
		}
		deleteUsers = append(deleteUsers, id)
	}

	// Add new posts.
	for i := 0; i < c.UpdatePosts && c.postIDs.len() < c.PostCount+c.UpdatePosts; i++ {
		id := c.postIDs.allocID()
		author, ok := c.userIDs.randomID()
		if !ok {
			break
		}
		log.Infof("[%s] add post %d by author %d", c, id, author)
		if err := c.addPost(id, author); err != nil {
			return errors.Trace(err)
		}
		newPosts = append(newPosts, id)
		newAuthors = append(newAuthors, author)
	}

	// Delete random posts.
	for i := 0; i < c.UpdatePosts && c.postIDs.len() > c.PostCount; i++ {
		id, ok := c.postIDs.randomID()
		if !ok {
			break
		}
		log.Infof("[%s] delete post %d", c, id)
		if err := c.deletePost(id); err != nil {
			return errors.Trace(err)
		}
		deletePosts = append(deletePosts, id)
	}

	// Check all.
	if err := c.checkAllPostCount(db); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (c *crudClient) createUser(db *sql.DB, id int64) error {
	c.userIDs.pushID(id)

	name := make([]byte, 10)
	util.RandString(name, c.rnd)
	_, err := util.ExecWithRollback(db, []util.QueryEntry{{Query: fmt.Sprintf(`INSERT INTO crud_users VALUES (%v, "%s", %v)`, id, name, 0), ExpectAffectedRows: 1}})
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c *crudClient) checkUserPostCount(id int64) error {
	var count1, count2 int64
	checkF := func() error {
		var err error
		count1, err = c.QueryInt64(fmt.Sprintf("SELECT posts FROM crud_users WHERE id=%v", id))
		if err != nil {
			return errors.Trace(err)
		}
		count2, err = c.QueryInt64(fmt.Sprintf("SELECT COUNT(*) FROM crud_posts WHERE author=%v", id))
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}
	if err := util.RunWithRetry(context.Background(), c.RetryLimit, 10*time.Second, checkF); err != nil {
		return errors.Trace(err)
	}

	if count1 != count2 {
		log.Fatalf("[%s] posts count not match %v != %v for user %v", c, count1, count2, id)
	}
	return nil
}

func (c *crudClient) deleteUser(id int64) error {
	if err := c.checkUserPostCount(id); err != nil {
		if errors.Cause(err) == errNotExist {
			c.userIDs.popID(id)
			return nil
		}
		return errors.Trace(err)
	}
	posts, err := c.QueryInt64s(fmt.Sprintf("SELECT id FROM crud_posts WHERE author=%v", id))
	if err != nil {
		return errors.Trace(err)
	}
	q := []util.QueryEntry{
		{Query: fmt.Sprintf("DELETE FROM crud_users WHERE id=%v", id), ExpectAffectedRows: 1},
		{Query: fmt.Sprintf("DELETE FROM crud_posts WHERE author=%v", id), ExpectAffectedRows: int64(len(posts))},
	}
	if _, err = util.ExecWithRollback(c.db, q); err != nil {
		return errors.Trace(err)
	}
	c.userIDs.popID(id)
	for _, id := range posts {
		c.postIDs.popID(id)
	}
	return nil
}

func (c *crudClient) addPost(id, author int64) error {
	if err := c.checkUserPostCount(author); err != nil {
		return errors.Trace(err)
	}
	c.postIDs.pushID(id)
	title := make([]byte, 64)
	util.RandString(title, c.rnd)
	q := []util.QueryEntry{
		{Query: fmt.Sprintf(`INSERT INTO crud_posts VALUES (%v, %v, "%s")`, id, author, title), ExpectAffectedRows: 1},
		{Query: fmt.Sprintf("UPDATE crud_users SET posts=posts+1 WHERE id=%v", author), ExpectAffectedRows: 1},
	}
	if _, err := util.ExecWithRollback(c.db, q); err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c *crudClient) deletePost(id int64) error {
	author, err := c.QueryInt64(fmt.Sprintf("SELECT author from crud_posts WHERE id=%v", id))
	if err != nil {
		if errors.Cause(err) == errNotExist {
			c.postIDs.popID(id)
			return nil
		}
		return errors.Trace(err)
	}
	if err := c.checkUserPostCount(author); err != nil {
		return errors.Trace(err)
	}
	q := []util.QueryEntry{
		{Query: fmt.Sprintf("DELETE FROM crud_posts WHERE id=%v", id), ExpectAffectedRows: 1},
		{Query: fmt.Sprintf("UPDATE crud_users SET posts=posts-1 WHERE id=%v", author), ExpectAffectedRows: 1},
	}
	if _, err := util.ExecWithRollback(c.db, q); err != nil {
		return errors.Trace(err)
	}
	c.postIDs.popID(id)
	return nil
}

func (c *crudClient) checkAllPostCount(db *sql.DB) error {
	var count1, count2 int64
	checkF := func() error {
		var err error
		count1, err = c.QueryInt64("SELECT SUM(posts) FROM crud_users")
		if err != nil {
			return errors.Trace(err)
		}
		count2, err = c.QueryInt64("SELECT COUNT(*) FROM crud_posts")
		if err != nil {
			return errors.Trace(err)
		}
		return nil
	}
	// read from tikv
	if _, err := db.Exec("set @@session.tidb_isolation_read_engines='tikv'"); err != nil {
		return err
	}
	if err := util.RunWithRetry(context.Background(), c.RetryLimit, 10*time.Second, checkF); err != nil {
		return errors.Trace(err)
	}
	if count1 != count2 {
		log.Fatalf("[%s] total posts count not match %v != %v", c, count1, count2)
	}
	// query with tiflash
	if c.TiFlashDataReplicas > 0 {
		if _, err := db.Exec("set @@session.tidb_isolation_read_engines='tiflash'"); err != nil {
			return err
		}

		if err := util.RunWithRetry(context.Background(), c.RetryLimit, 10*time.Second, checkF); err != nil {
			return errors.Trace(err)
		}
		if count1 != count2 {
			log.Fatalf("[%s] total posts count not match %v != %v", c, count1, count2)
		}

		if _, err := db.Exec("set @@session.tidb_isolation_read_engines='tikv'"); err != nil {
			return err
		}
	}

	return nil
}

// QueryInt64s queries int64s
func (c *crudClient) QueryInt64s(query string, args ...interface{}) ([]int64, error) {
	var vals []int64

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return []int64{}, errors.Trace(err)
	}
	defer rows.Close()
	for rows.Next() {
		var val int64
		if err := rows.Scan(&val); err != nil {
			log.Fatalf("[%s] failed to scan int64 result: %v", c, err)
		}
		vals = append(vals, val)
	}
	if err := rows.Err(); err != nil {
		return []int64{}, errors.Trace(err)
	}
	return vals, errors.Trace(err)
}

// QueryInt64 query int64
func (c *crudClient) QueryInt64(query string, args ...interface{}) (int64, error) {
	vals, err := c.QueryInt64s(query, args...)
	if err != nil {
		return 0, err
	}
	if len(vals) == 0 {
		return 0, errNotExist
	}
	if len(vals) != 1 {
		return 0, errors.Errorf("expect 1 row for query %v, but got %v rows", query, len(vals))
	}
	return vals[0], nil
}
