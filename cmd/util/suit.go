package util

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
	core.ClientCreator
	// nemesis, separated by comma.
	Nemesises string
	// perform service quality checking
	WithProf   bool
	VerifySuit verify.Suit

	// cluster definition
	Cluster interface{}
}

// Run runs the suit.
func (suit *Suit) Run(ctx context.Context) {
	var err error
	var nemesisGens []core.NemesisGenerator
	for _, name := range strings.Split(suit.Nemesises, ",") {
		var g core.NemesisGenerator
		name := strings.TrimSpace(name)
		if len(name) == 0 {
			continue
		}

		switch name {
		case "random_kill", "all_kill", "minor_kill", "major_kill",
			"kill_tikv_1node_5min", "kill_tikv_2node_5min",
			"kill_pd_leader_5min", "kill_pd_nonleader_5min":
			g = nemesis.NewKillGenerator(name)
		case "short_kill_tikv_1node", "short_kill_pd_leader":
			g = nemesis.NewContainerKillGenerator(name)
		case "random_drop", "all_drop", "minor_drop", "major_drop":
			log.Fatal("Unimplemented")
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

		nemesisGens = append(nemesisGens, g)
	}

	sctx, cancel := context.WithCancel(ctx)

	suit.Config.Nodes, suit.Config.ClientNodes, err = suit.Provisioner.SetUp(sctx, suit.Cluster)

	if err != nil {
		log.Fatalf("deploy a cluster failed, err: %s", err)
	}
	log.Printf("deploy cluster success, node:%+v, client node:%+v", suit.Config.Nodes, suit.Config.ClientNodes)
	if len(suit.Config.ClientNodes) == 0 {
		log.Panic("no client nodes exist")
	}
	suit.Config.ClientCount = len(suit.Config.ClientNodes)
	if suit.Config.ClientCount == 0 {
		log.Panic("suit.Config.ClientCount is required")
	}
	// fill clientNodes
	retClientCount := len(suit.Config.ClientNodes)
	for len(suit.Config.ClientNodes) < suit.Config.ClientCount {
		suit.Config.ClientNodes = append(suit.Config.ClientNodes,
			suit.Config.ClientNodes[rand.Intn(retClientCount)])
	}

	// sleep 10s to make sure nodeport works and tidb is ready
	time.Sleep(10 * time.Second)

	c := control.NewController(
		sctx,
		suit.Config,
		suit.ClientCreator,
		nemesisGens,
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
}
