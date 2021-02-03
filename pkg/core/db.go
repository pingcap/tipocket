package core

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
)

// DB allows to set up and tear down database.
type DB interface {
	// SetUp initializes the database.
	SetUp(ctx context.Context, nodes []cluster.Node, node cluster.Node) error
	// TearDown tears down the database.
	TearDown(ctx context.Context, nodes []cluster.Node, node cluster.Node) error
	// Name returns the unique name for the database
	Name() string
}

// NoopDB is a DB but does nothing
type NoopDB struct {
}

// SetUp initializes the database.
func (NoopDB) SetUp(ctx context.Context, nodes []cluster.Node, node cluster.Node) error {
	// pre-set global variables
	if len(strings.TrimSpace(fixture.Context.TiDBClusterConfig.PrepareSQL)) == 0 {
		return nil
	}
	isFirstTiDB := false
	for _, n := range nodes {
		if n.Component == cluster.TiDB {
			isFirstTiDB = n == node
			break
		}
	}

	// Run prepare SQLs
	if isFirstTiDB {
		db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(%s:%d)/test?multiStatements=true", node.IP, node.Port))
		if err != nil {
			return err
		}
		if _, err = db.Exec(fixture.Context.TiDBClusterConfig.PrepareSQL); err != nil {
			return err
		}
		if err = db.Close(); err != nil {
			return err
		}
		// Wait until global variables take effect
		time.Sleep(2100 * time.Millisecond)
	}
	return nil
}

// TearDown tears down the database.
func (NoopDB) TearDown(ctx context.Context, nodes []cluster.Node, node cluster.Node) error {
	return nil
}

// Name returns the unique name for the database
func (NoopDB) Name() string {
	return ""
}

var dbs = map[string]DB{}

// RegisterDB registers db. Not thread-safe
func RegisterDB(db DB) {
	name := db.Name()
	_, ok := dbs[name]
	if ok {
		panic(fmt.Sprintf("db %s is already registered", name))
	}

	dbs[name] = db
}

// GetDB gets the registered db.
func GetDB(name string) DB {
	return dbs[name]
}

func init() {
	RegisterDB(NoopDB{})
}
