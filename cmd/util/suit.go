package util

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/nemesis"
	"github.com/pingcap/tipocket/pkg/verify"
)

// Suit is a basic chaos testing suit with configurations to run chaos.
type Suit struct {
	*control.Config
	// Provisioner deploy the SUT cluster
	clusterTypes.Provisioner
	// ClientCreator creates client
	core.ClientCreator
	// NemesisGens saves NemesisGenerator
	NemesisGens []core.NemesisGenerator
	// perform service quality checking
	VerifySuit verify.Suit
	// cluster definition
	ClusterDefs interface{}
}

// Run runs the suit.
func (suit *Suit) Run(ctx context.Context) {
	var err error
	sctx, cancel := context.WithCancel(ctx)
	suit.Config.Nodes, suit.Config.ClientNodes, err = suit.Provisioner.SetUp(sctx, suit.ClusterDefs)
	if err != nil {
		log.Fatalf("deploy a cluster failed, err: %s", err)
	}
	log.Printf("deploy cluster success, node:%+v, client node:%+v", suit.Config.Nodes, suit.Config.ClientNodes)
	if len(suit.Config.ClientNodes) == 0 {
		log.Panic("no client nodes exist")
	}
	if suit.Config.ClientCount == 0 {
		log.Panic("suit.Config.ClientCount is required")
	}
	// fill clientNodes
	retClientCount := len(suit.Config.ClientNodes)
	for len(suit.Config.ClientNodes) < suit.Config.ClientCount {
		suit.Config.ClientNodes = append(suit.Config.ClientNodes,
			suit.Config.ClientNodes[rand.Intn(retClientCount)])
	}
	c := control.NewController(
		sctx,
		suit.Config,
		suit.ClientCreator,
		suit.NemesisGens,
		suit.VerifySuit,
	)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		<-sigs
		c.Close()
		cancel()
	}()

	c.Run()

	log.Printf("tear down cluster...")
	if err := suit.Provisioner.TearDown(context.TODO(), suit.ClusterDefs); err != nil {
		log.Printf("Provisioner tear down failed: %+v", err)
	}
}

// ParseNemesisGenerator parse NemesisGenerator from string literal
func ParseNemesisGenerator(name string) (g core.NemesisGenerator) {
	switch name {
	case "random_kill", "all_kill", "minor_kill", "major_kill",
		"kill_tikv_1node_5min", "kill_tikv_2node_5min",
		"kill_pd_leader_5min", "kill_pd_nonleader_5min":
		g = nemesis.NewKillGenerator(name)
	case "short_kill_tikv_1node", "short_kill_pd_leader":
		g = nemesis.NewContainerKillGenerator(name)
	case "random_drop", "all_drop", "minor_drop", "major_drop":
		log.Panic("Unimplemented")
	case "partition_one":
		g = nemesis.NewNetworkPartitionGenerator(name)
	case "loss", "delay", "duplicate", "corrupt":
		g = nemesis.NewNetemChaos(name)
	case "pod_kill":
		g = nemesis.NewPodKillGenerator(name)
	case "noop":
		g = core.NoopNemesisGenerator{}
	default:
		log.Fatalf("invalid nemesis generator %s", name)
	}
	return
}

// ParseNemesisGenerators, names are separated by ","
func ParseNemesisGenerators(names string) (nemesisGens []core.NemesisGenerator) {
	for _, name := range strings.Split(names, ",") {
		name := strings.TrimSpace(name)
		if len(name) == 0 {
			continue
		}
		nemesisGens = append(nemesisGens, ParseNemesisGenerator(name))
	}
	return
}
