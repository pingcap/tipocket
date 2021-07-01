package control

import (
	"time"

	"github.com/pingcap/tipocket/pkg/cluster"
)

// Config is the configuration for the controller.
type Config struct {
	// Mode is for switch between control modes
	Mode Mode
	// DB is the name which we want to run.
	DB string
	// Nodes are addresses of nodes.
	Nodes []cluster.Node
	// ClientNode are addresses of client usage.
	ClientNodes []cluster.ClientNode
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

	// ClientConfig can be anything, use type assertion in your case
	ClientConfig interface{}
}

// Mode enums control modes
type Mode int

// Modes enum
const (
	ModeOnSchedule = iota
	ModeStandard
	ModeNemesisSequential
)

func (m Mode) String() string {
	switch m {
	case ModeOnSchedule:
		return "On Schedule mode"
	case ModeStandard:
		return "Standard or default mode"
	case ModeNemesisSequential:
		return "Nemesis one by one mode"
	default:
		return "Unknown mode"
	}
}
