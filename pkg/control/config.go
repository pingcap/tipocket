package control

import (
	"time"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
)

// Config is the configuration for the controller.
type Config struct {
	// Mode is for switch between control modes
	Mode Mode
	// DB is the name which we want to run.
	DB core.DB
	// Nodes are addresses of nodes.
	Nodes []clusterTypes.Node
	// ClientNode are addresses of client usage.
	ClientNodes []clusterTypes.ClientNode
	// ClientCount controls the count of clients
	ClientCount int
	// Chaos NS
	ChaosNamespace string

	// RunRound controls how many round the controller runs tests.
	RunRound int
	// RunTime controls how long a round takes.
	RunTime time.Duration
	// RequestCount controls how many requests a client sends to the db.
	RequestCount int

	// History file
	History string

	// CaseConfig can be anything, use type assertion in your case
	CaseConfig interface{}
}

// Mode enums control modes
type Mode int

// Modes enum
const (
	ModeMixed = iota
	ModeSequential
	ModeSelfScheduled
)

func (m Mode) String() string {
	switch m {
	case ModeMixed:
		return "Mixed mode"
	case ModeSelfScheduled:
		return "Self scheduled mode"
	case ModeSequential:
		return "Nemesis one by one mode"
	default:
		return "Unknown mode"
	}
}

func (c *Config) adjust() {
	if c.DB == nil {
		c.DB = core.NoopDB{}
	}
	if c.RequestCount == 0 {
		c.RequestCount = 10000
	}

	if c.RunTime == 0 {
		c.RunTime = 10 * time.Minute
	}

	if c.RunRound == 0 {
		c.RunRound = 20
	}
}
