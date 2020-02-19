package creator

import (
	"context"
	"fmt"

	"github.com/pingcap/errors"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
	"github.com/pingcap/tipocket/pkg/core"
	"github.com/pingcap/tipocket/pkg/pocket/config"
	pocketCore "github.com/pingcap/tipocket/pkg/pocket/core"
)

// PocketCreator create pocket instances
type PocketCreator struct{}

// PocketClient runs pocket
type PocketClient struct{}

// Create client
func (PocketCreator) Create(node clusterTypes.ClientNode) core.Client {
	return PocketClient{}
}

// SetUp sets up the client.
func (PocketClient) SetUp(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error {
	return nil
}

// TearDown tears down the client.
func (PocketClient) TearDown(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error {
	return nil
}

// Invoke invokes a request to the database.
func (PocketClient) Invoke(ctx context.Context, node clusterTypes.ClientNode, r interface{}) interface{} {
	return nil
}

// NextRequest generates a request for latter Invoke.
func (PocketClient) NextRequest() interface{} {
	return nil
}

// DumpState the database state(also the model's state)
func (PocketClient) DumpState(ctx context.Context) (interface{}, error) {
	return nil, nil
}

// Start runs self scheduled cases
func (PocketClient) Start(ctx context.Context, caseConfig interface{}, clientNodes []clusterTypes.ClientNode) error {
	var (
		upstream, downstream = makeDSN(clientNodes[0].String()), makeDSN(clientNodes[1].String())
		cfgPath              = caseConfig.(string)
	)

	cfg := config.Init()

	if err := cfg.Load(cfgPath); err != nil {
		return errors.Trace(err)
	}

	cfg.Mode = "binlog"
	cfg.DSN1, cfg.DSN2 = upstream, downstream

	return pocketCore.New(cfg).Start(ctx)
}

func makeDSN(address string) string {
	return fmt.Sprintf("root:@tcp(%s)/pocket", address)
}
