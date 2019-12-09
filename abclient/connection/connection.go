package connection

import (
	"fmt"
	"github.com/pingcap/tipocket/abclient/pkg/logger"
	"github.com/pingcap/tipocket/abclient/pkg/mysql"
	"github.com/juju/errors"
)

// Option struct
type Option struct {
	Log string
	Mute bool
}

// Connection define connection struct
type Connection struct {
	logger *logger.Logger
	db *mysql.DBConnect
}

// New create Connection instance from dsn
func New(dsn string, opt *Option) (*Connection, error) {
	l, err := logger.New(opt.Log, opt.Mute)
	if err != nil {
		return nil, errors.Trace(err)
	}
	db, err := mysql.OpenDB(dsn, 1)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &Connection{
		logger: l,
		db: db,
	}, nil
}

// Prepare create test database
func (c *Connection) Prepare(db string) {
	c.db.MustExec(fmt.Sprintf(dropDatabaseSQL, db))
	c.db.MustExec(fmt.Sprintf(createDatabaseSQL, db))
}

// CloseDB close connection
func (c *Connection) CloseDB() error {
	return c.db.CloseDB()
}

// ReConnect rebuild connection
func (c *Connection) ReConnect() error {
	return c.db.ReConnect()
}
