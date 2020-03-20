package core

import (
	"context"

	clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"
)

// UnknownResponse means we don't know wether this operation
// succeeds or not.
type UnknownResponse interface {
	IsUnknown() bool
}

// Client applies the request to the database.
// Client is used in control.
// You should define your own client for your database.
type Client interface {
	// SetUp sets up the client.
	SetUp(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error
	// TearDown tears down the client.
	TearDown(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error
	// Invoke invokes a request to the database.
	// Mostly, the return Response should implement UnknownResponse interface
	Invoke(ctx context.Context, node clusterTypes.ClientNode, r interface{}) interface{}
	// NextRequest generates a request for latter Invoke.
	NextRequest() interface{}
	// DumpState the database state(also the model's state)
	DumpState(ctx context.Context) (interface{}, error)
	// Start runs self scheduled cases
	// this function will block Invoke trigger
	// if you want to schedule cases by yourself, use this function only
	Start(ctx context.Context, cfg interface{}, clientNodes []clusterTypes.ClientNode) error
}

// ClientCreator creates a client.
// The control will create one client for one node.
type ClientCreator interface {
	// Create creates the client.
	Create(node clusterTypes.ClientNode) Client
}

// NoopClientCreator creates a noop client.
type NoopClientCreator struct {
}

// Create creates the client.
func (NoopClientCreator) Create(node clusterTypes.Node) Client {
	return noopClient{}
}

// noopClient is a noop client
type noopClient struct {
}

// SetUp sets up the client.
func (noopClient) SetUp(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error {
	return nil
}

// TearDown tears down the client.
func (noopClient) TearDown(ctx context.Context, nodes []clusterTypes.ClientNode, idx int) error {
	return nil
}

// Invoke invokes a request to the database.
func (noopClient) Invoke(ctx context.Context, node clusterTypes.ClientNode, r interface{}) interface{} {
	return nil
}

// NextRequest generates a request for latter Invoke.
func (noopClient) NextRequest() interface{} {
	return nil
}

// DumpState the database state(also the model's state)
func (noopClient) DumpState(ctx context.Context) (interface{}, error) {
	return nil, nil
}

// Start runs self scheduled cases
func (noopClient) Start(ctx context.Context, cfg interface{}, clientNodes []clusterTypes.ClientNode) error {
	return nil
}
