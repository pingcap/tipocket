package executor

import (
	"fmt"

	"github.com/pingcap/tipocket/pocket/connection"
)

// IfTxn show if in a transaction
func (e *Executor) IfTxn() bool {
	switch e.mode {
	case "single":
		return e.singleTestIfTxn()
	case "abtest":
		return e.abTestIfTxn()
	}
	panic("unhandled switch")
}

// GetConn get connection of first connection
func (e *Executor) GetConn() *connection.Connection {
	return e.GetConn1()
}

// GetConn1 get connection of first connection
func (e *Executor) GetConn1() *connection.Connection {
	return e.conn1
}

// GetConn2 get connection of second connection
func (e *Executor) GetConn2() *connection.Connection {
	return e.conn2
}

// ReConnect rebuild connection
func (e *Executor) ReConnect() error {
	switch e.mode {
	case "single":
		return e.conn1.ReConnect()
	case "abtest":
		if err := e.conn1.ReConnect(); err != nil {
			return err
		}
		return e.conn2.ReConnect()
	}
	panic("unhandled reconnect switch")
}

// Close close connection
func (e *Executor) Close() error {
	switch e.mode {
	case "single":
		return e.conn1.CloseDB()
	case "abtest":
		err1 := e.conn1.CloseDB()
		err2 := e.conn2.CloseDB()
		if err1 != nil {
			return err1
		}
		return err2
	}
	panic("unhandled reconnect switch")
}

// Exec function for quick executor some SQLs
func (e *Executor) Exec(sql string) error {
	switch e.mode {
	case "single":
		if err := e.conn1.Exec(sql); err != nil {
			return err
		}
		return e.conn1.Commit()
	case "abtest":
		if err := e.conn1.Exec(sql); err != nil {
			return err
		}
		if err := e.conn1.Commit(); err != nil {
			return err
		}
		if err := e.conn2.Exec(sql); err != nil {
			return err
		}
		return e.conn2.Commit()
	}
	panic("unhandled reconnect switch")
}

// ExecIgnoreErr function for quick executor some SQLs with error tolerance
func (e *Executor) ExecIgnoreErr(sql string) {
	switch e.mode {
	case "single":
		_ = e.conn1.Exec(sql)
		_ = e.conn1.Commit()
	case "abtest", "binlog":
		_ = e.conn1.Exec(sql)
		_ = e.conn1.Commit()
		_ = e.conn2.Exec(sql)
		_ = e.conn2.Commit()
	default:
		panic(fmt.Sprintf("unhandled reconnect switch , mode: %s", e.mode))
	}
}
