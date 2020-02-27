package core

import (
	"context"
	"fmt"
	"time"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
)

// ChaosKind is Kind of applying chaos
type ChaosKind string

const (
	// PodFailure Apply pod failure
	PodFailure ChaosKind = "Pod-Failure"
	// PodKill will random kill a pod, this will make the Node be illegal
	PodKill ChaosKind = "Pod-Kill"
	// ContainerKill will random kill the specified container of pod, but retain the pod
	ContainerKill ChaosKind = "Container-Kill"
	// NetworkPartition parts network between nodes
	NetworkPartition ChaosKind = "Network-Partition"
	// NetemChaos add corrupt or other chaos.
	NetemChaos ChaosKind = "Netem-Chaos"
)

// Nemesis injects failure and disturbs the database.
type Nemesis interface {
	// // SetUp initializes the nemesis
	// SetUp(ctx context.Context, node string) error
	// // TearDown tears down the nemesis
	// TearDown(ctx context.Context, node string) error

	// Invoke executes the nemesis
	Invoke(ctx context.Context, node *clusterTypes.Node, args ...interface{}) error
	// Recover recovers the nemesis
	Recover(ctx context.Context, node *clusterTypes.Node, args ...interface{}) error
	// Name returns the unique name for the nemesis
	Name() string
}

// NoopNemesis implements Nemesis but does nothing
type NoopNemesis struct {
}

// // SetUp initializes the nemesis
// func (NoopNemesis) SetUp(ctx context.Context, node string) error {
// 	return nil
// }

// // TearDown tears down the nemesis
// func (NoopNemesis) TearDown(ctx context.Context, node string) error {
// 	return nil
// }

// Invoke executes the nemesis
func (NoopNemesis) Invoke(ctx context.Context, node *clusterTypes.Node, args ...interface{}) error {
	return nil
}

// Recover recovers the nemesis
func (NoopNemesis) Recover(ctx context.Context, node *clusterTypes.Node, args ...interface{}) error {
	return nil
}

// Name returns the unique name for the nemesis
func (NoopNemesis) Name() string {
	return "noop"
}

var nemesises = map[string]Nemesis{}

// RegisterNemesis registers nemesis. Not thread-safe.
func RegisterNemesis(n Nemesis) {
	name := n.Name()
	_, ok := nemesises[name]
	if ok {
		panic(fmt.Sprintf("nemesis %s is already registered", name))
	}

	nemesises[name] = n
}

// GetNemesis gets the registered nemesis.
func GetNemesis(name string) Nemesis {
	return nemesises[name]
}

// NemesisOperation is nemesis operation used in control.
type NemesisOperation struct {
	Type        ChaosKind          // Nemesis name
	Node        *clusterTypes.Node // Nemesis target node, optional if it affects
	InvokeArgs  []interface{}      // Nemesis invoke args
	RecoverArgs []interface{}      // Nemesis recover args
	RunTime     time.Duration      // Nemesis duration time
}

// NemesisGeneratorRecord is used to record operations generated by NemesisGenerator.Generate
type NemesisGeneratorRecord struct {
	Name string
	Ops  []*NemesisOperation
}

// NemesisGenerator is used in control, it will generate a nemesis operation
// and then the control can use it to disturb the cluster.
type NemesisGenerator interface {
	// Generate generates the nemesis operation for all nodes.
	// Every node will be assigned a nemesis operation.
	Generate(nodes []clusterTypes.Node) []*NemesisOperation
	Name() string
}

// NoopNemesisGenerator generates
type NoopNemesisGenerator struct {
}

// Name returns the name
func (NoopNemesisGenerator) Name() string {
	return "noop"
}

//Generate generates the nemesis operation for the nodes.
func (NoopNemesisGenerator) Generate(nodes []clusterTypes.Node) []*NemesisOperation {
	ops := make([]*NemesisOperation, len(nodes))
	for i := 0; i < len(ops); i++ {
		ops[i] = &NemesisOperation{
			// noop do nothing
			Type:        "noop",
			Node:        &nodes[i],
			InvokeArgs:  nil,
			RecoverArgs: nil,
			RunTime:     0,
		}
	}
	return ops
}

type DelayNemesisGenerator struct {
	Gen   NemesisGenerator
	Delay time.Duration
}

func (d DelayNemesisGenerator) Generate(nodes []clusterTypes.Node) []*NemesisOperation {
	time.Sleep(d.Delay)
	return d.Gen.Generate(nodes)
}

func (d DelayNemesisGenerator) Name() string {
	return d.Gen.Name()
}

func init() {
	RegisterNemesis(NoopNemesis{})
}
