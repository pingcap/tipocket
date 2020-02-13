package nemesis

import (
	"math/rand"
	"time"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
)

type killGenerator struct {
	name string
}

func (g killGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	n := 1

	// This part decide how many machines to apply pod-failure
	switch g.name {
	case "minor_kill":
		n = len(nodes)/2 - 1
	case "major_kill":
		n = len(nodes)/2 + 1
	case "all_kill":
		n = len(nodes)
	default:
		n = 1
	}

	return killNodes(nodes, n)
}

func (g killGenerator) Name() string {
	return string(core.PodFailure)
}

func killNodes(nodes []cluster.Node, n int) []*core.NemesisOperation {
	ops := make([]*core.NemesisOperation, len(nodes))

	// randomly shuffle the indices and get the first n nodes to be partitioned.
	indices := shuffleIndices(len(nodes))

	for i := 0; i < n; i++ {
		ops[indices[i]] = &core.NemesisOperation{
			Type: core.PodFailure,
			// Note: Maybe I should just store cluster info here.
			InvokeArgs:  nil,
			RecoverArgs: nil,
			RunTime:     time.Second * time.Duration(rand.Intn(120)+10),
		}
	}

	return ops
}

// NewKillGenerator creates a generator.
// Name is random_kill, minor_kill, major_kill, and all_kill.
func NewKillGenerator(name string) core.NemesisGenerator {
	return killGenerator{name: name}
}

type dropGenerator struct {
	name string
}

func (g dropGenerator) Generate(nodes []cluster.Node) []*core.NemesisOperation {
	n := 1
	switch g.name {
	case "minor_drop":
		n = len(nodes)/2 - 1
	case "major_drop":
		n = len(nodes)/2 + 1
	case "all_drop":
		n = len(nodes)
	default:
		n = 1
	}
	return partitionNodes(nodes, n)
}

func (g dropGenerator) Name() string {
	return g.name
}

// TODO: this code was still non-k8s style.
func partitionNodes(nodes []cluster.Node, n int) []*core.NemesisOperation {
	ops := make([]*core.NemesisOperation, len(nodes))

	// randomly shuffle the indices and get the first n nodes to be partitioned.
	indices := shuffleIndices(len(nodes))

	partNodes := make([]string, n)
	for i := 0; i < n; i++ {
		partNodes[i] = nodes[indices[i]].IP
	}

	for i := 0; i < len(nodes); i++ {
		ops[i] = &core.NemesisOperation{
			Type:       "drop",
			InvokeArgs: partNodes,
			RunTime:    time.Second * time.Duration(rand.Intn(120)+10),
		}
	}

	return ops
}

func shuffleIndices(n int) []int {
	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
	}
	for i := len(indices) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		indices[i], indices[j] = indices[j], indices[i]
	}

	return indices
}

// NewDropGenerator creates a generator.
// Name is random_drop, minor_drop, major_drop, and all_drop.
func NewDropGenerator(name string) core.NemesisGenerator {
	return dropGenerator{name: name}
}
