package core

import (
	"context"
	"fmt"
	"time"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
)

// ChaosKind is the kind of applying chaos
type ChaosKind string

const (
	// PodFailure Applies pod failure
	PodFailure ChaosKind = "pod-failure"
	// PodKill will random kill a pod, this will make the Node be illegal
	PodKill ChaosKind = "pod-kill"
	// ContainerKill will random kill the specified container of pod, but retain the pod
	ContainerKill ChaosKind = "container-kill"
	// NetworkPartition partitions network between nodes
	NetworkPartition ChaosKind = "network-partition"
	// NetemChaos adds corrupt or other chaos.
	NetemChaos ChaosKind = "netem-chaos"
	// TimeChaos means
	TimeChaos ChaosKind = "time-chaos"
	// PDScheduler adds scheduler
	PDScheduler ChaosKind = "PDConfig-Scheduler"
	// PDLeaderShuffler will randomly shuffle pds.
	PDLeaderShuffler ChaosKind = "PDConfig-Leader-Shuffler"
	// Scaling scales cluster
	Scaling ChaosKind = "scaling"
	// IOChaos adds io chaos.
	IOChaos ChaosKind = "io-chaos"
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

// String ...
func (n NemesisOperation) String() string {
	return fmt.Sprintf("type:%s,duration:%s,node:%s,invoke_args:%+v,recover_args:%+v", n.Type, n.RunTime, n.Node, n.InvokeArgs, n.RecoverArgs)
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

// DelayNemesisGenerator delays nemesis generation after `Delay`
type DelayNemesisGenerator struct {
	Gen   NemesisGenerator
	Delay time.Duration
}

// Generate ...
func (d DelayNemesisGenerator) Generate(nodes []clusterTypes.Node) []*NemesisOperation {
	time.Sleep(d.Delay)
	return d.Gen.Generate(nodes)
}

// Name ...
func (d DelayNemesisGenerator) Name() string {
	return d.Gen.Name()
}
