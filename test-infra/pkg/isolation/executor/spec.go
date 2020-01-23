package executor

import (
	"github.com/pingcap/tipocket/pocket/pkg/mysql"
	"github.com/pingcap/tipocket/test-infra/pkg/isolation/model"
)

type Executor struct {
	dsn string

	conn []*mysql.DBConnect
}

type WaitLogConnect struct {
	conn mysql.DBConnect
}

// TODO: add extra checking for executor dsn
func NewExecutor(dsn string) (*Executor, error) {
	return &Executor{dsn: dsn}, nil
}

func (e *Executor) open(cnt int) error {
	for i := 0; i < cnt; i++ {
		newConn, err := mysql.OpenDB(e.dsn, 1)
		if err != nil {
			return err
		}
		e.conn = append(e.conn, newConn)
	}
}

func (e *Executor) Close() {
	for i := 0; i < len(e.conn); i++ {
		_ = e.conn[i].CloseDB()
	}
	e.conn = nil
}

func (e *Executor) Execute(spec model.TestSpec) (bool, error) {
	// first phases runs initialize scripts
	err := e.open(len(spec.Sessions) + 1)
	if err != nil {
		return false, err
	}
	// Prepare Phase: Execute initialize connection
	initConn := e.conn[0]
	for _, setup := range spec.SetupSQLs {
		initConn.MustExec(setup)
	}

	// Prepare Phase: run all setupSQL in sub-session
	for _, session := range spec.Sessions {
		if session.SetupSQL != "" {
			initConn.MustExec(session.SetupSQL)
		}
	}

	defer func() {
		defer e.Close()

		initConn := e.conn[0]

		// Prepare Phase: run all setupSQL in sub-session
		for _, session := range spec.Sessions {
			if session.TeardownSQL != "" {
				initConn.MustExec(session.SetupSQL)
			}
		}

		initConn.MustExec(spec.TeardownSQL)
	}()

	panic("impl me")
}

func (e *Executor) RunSpec(spec *model.TestSpec) {
	panic("impl me")
}

func (e *Executor) RunPermutations(perm *model.Permutation) {
	panic("impl me")
}

func (e *Executor) Diff(perm *model.Permutation) {
	panic("impl me")
}

type Execution interface {
	Execute(spec model.TestSpec) bool
}
