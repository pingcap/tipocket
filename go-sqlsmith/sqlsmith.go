package sqlsmith

import (
	"math/rand"
	"time"

	"github.com/juju/errors"
	"github.com/pingcap/parser/ast"
	"github.com/pingcap/tipocket/go-sqlsmith/stateflow"
	"github.com/pingcap/tipocket/go-sqlsmith/types"
	"github.com/pingcap/tipocket/go-sqlsmith/util"

	// _ "github.com/pingcap/tidb/types/parser_driver"
)

// SQLSmith defines SQLSmith struct
type SQLSmith struct {
	depth int
	maxDepth int
	Rand *rand.Rand
	Databases map[string]*types.Database
	subTableIndex int
	Node ast.Node
	currDB string
	debug bool
	stable bool
}

// New create SQLSmith instance
func New() *SQLSmith {
	return &SQLSmith{
		Rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
		Databases: make(map[string]*types.Database),
	}
}

// Debug turn on debug mode
func (s *SQLSmith) Debug() {
	s.debug = true
}

// SetDB set current database
func (s *SQLSmith) SetDB(db string) {
	s.currDB = db
}

// GetCurrDBName returns current selected dbname
func (s *SQLSmith) GetCurrDBName() string {
	return s.currDB
}

// GetDB get current database without nil
func (s *SQLSmith) GetDB(db string) *types.Database {
	if db, ok := s.Databases[db]; ok {
		return db
	}
	return &types.Database{
		Name: db,
	}
}

// Stable set generated SQLs no rand
func (s *SQLSmith) Stable() {
	s.stable = true
}

// SetStable set stable to given value
func (s *SQLSmith) SetStable(stable bool) {
	s.stable = stable
}

// Walk will walk the tree and fillin tables and columns data
func (s *SQLSmith) Walk(tree ast.Node) (string, error) {
	node := stateflow.New(s.GetDB(s.currDB), s.stable).WalkTree(tree)
	s.debugPrintf("node AST %+v\n", node)
	sql, err := util.BufferOut(node)
	// if sql ==
	return sql, errors.Trace(err)
}
