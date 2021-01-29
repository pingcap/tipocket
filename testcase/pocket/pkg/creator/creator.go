package creator

import (
	"context"
	"fmt"

	"github.com/pingcap/errors"

	"github.com/pingcap/tipocket/pkg/cluster"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/testcase/pocket/pkg/config"
	pocketCore "github.com/pingcap/tipocket/testcase/pocket/pkg/core"
)

// Config struct
type Config struct {
	*config.Config
	ConfigPath string
	Mode       string
}

// PocketCreator create pocket instances
type PocketCreator struct {
	Config
}

// PocketClient runs pocket
type PocketClient struct {
	Config
}

// Create client
func (p PocketCreator) Create(node cluster.ClientNode) core.Client {
	return PocketClient{p.Config}
}

// SetUp sets up the client.
func (PocketClient) SetUp(ctx context.Context, _ []cluster.Node, _ []cluster.ClientNode, idx int) error {
	return nil
}

// TearDown tears down the client.
func (PocketClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

// Start runs auto driver cases
func (p PocketClient) Start(ctx context.Context, _ interface{}, clientNodes []cluster.ClientNode) error {
	var cfgPath = p.Config.ConfigPath

	cfg := p.Config.Config
	if cfgPath != "" {
		if err := cfg.Load(cfgPath); err != nil {
			return errors.Trace(err)
		}
	}

	cfg.Mode = p.Config.Mode
	cfg.DSN1 = makeDSN(clientNodes[0].Address())

	// In the TiFlash case, there is only one client node.
	if len(clientNodes) > 1 {
		cfg.DSN2 = makeDSN(clientNodes[1].Address())
	}
	// In the DM case, there are 3 client nodes needed (2 for upstream MySQL and 1 for downstream TiDB).
	if len(clientNodes) > 2 {
		cfg.DSN3 = makeDSN(clientNodes[2].Address())
	}

	pCore := pocketCore.New(cfg)

	if cfg.Mode == "dm" {
		err := dmCreateSourceTask(clientNodes)
		if err != nil {
			return err
		}

		err = dmSyncDiffData(ctx, pCore, cfg.Options.CheckDuration.Duration, clientNodes)
		if err != nil {
			return err
		}
	}

	return pCore.Start(ctx)
}

func makeDSN(address string) string {
	return fmt.Sprintf("root:@tcp(%s)/pocket", address)
}
