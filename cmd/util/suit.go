package util

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/pingcap/tipocket/pkg/cluster"

	"github.com/pingcap/tipocket/pkg/control"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/nemesis"
	"github.com/pingcap/tipocket/pkg/verify"
)

// Suit is a basic chaos testing suit with configurations to run chaos.
type Suit struct {
	*control.Config
	// Provisioner deploy the SUT cluster
	cluster.Provisioner
	core.ClientCreator
	// nemesis, seperated by comma.
	Nemesises string

	VerifySuit verify.Suit

	// cluster definition
	Cluster interface{}

	// The namespace where chaos-mesh deployed.
	ChaosNamespace string
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
		case "random_kill", "all_kill", "minor_kill", "major_kill":
			g = nemesis.NewKillGenerator(name)
		case "random_drop", "all_drop", "minor_drop", "major_drop":
			log.Fatal("Unimplemented")
			g = nemesis.NewDropGenerator(name)
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
	log.Println("deploy cluster success")

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
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		c.Close()
		cancel()
	}()

	c.Run()
}
