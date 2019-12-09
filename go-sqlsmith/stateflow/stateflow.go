package stateflow

import (
	"math/rand"
	"time"
	"github.com/pingcap/tipocket/go-sqlsmith/types"
)

// StateFlow defines the struct
type StateFlow struct {
	// db is a ref took from SQLSmith, should never change it
	db         *types.Database
	rand       *rand.Rand
	tableIndex int
	stable     bool
}

// New Create StateFlow
func New(db *types.Database, stable bool) *StateFlow {
	// copy whole db here may cost time, but ensure the global safety
	// maybe a future TODO
	return &StateFlow{
		db: db,
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
		stable: stable,
	}
}
