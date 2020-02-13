package control

import (
	"time"

	"github.com/pingcap/tipocket/pkg/cluster"
)

// Config is the configuration for the controller.
type Config struct {
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
}

func (c *Config) adjust() {
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
