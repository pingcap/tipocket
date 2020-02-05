package util

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

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
			g = nemesis.NewKillGenerator(suit.Config.DB, name)
		case "random_drop", "all_drop", "minor_drop", "major_drop":
			g = nemesis.NewDropGenerator(name)
		default:
			log.Fatalf("invalid nemesis generator %s", name)
		}

		nemesisGens = append(nemesisGens, g)
	}

	sctx, cancel := context.WithCancel(ctx)

	err, suit.Config.Nodes = suit.Provisioner.SetUp(sctx, nil)
	if err != nil {
		log.Fatalf("deploy a cluster failed, err: %s", err)
	}

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
