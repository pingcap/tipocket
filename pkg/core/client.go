package core

import (
	"context"

	"github.com/pingcap/tipocket/pkg/cluster"
)

// UnknownResponse means we don't know whether this operation
// succeeds or not.
type UnknownResponse interface {
	IsUnknown() bool
}

// Client applies the request to the database.
// Client is used in control.
// You should define your own client for your database.
type Client interface {
	// SetUp sets up the client.
	SetUp(ctx context.Context, nodes []cluster.Node, clientNodes []cluster.ClientNode, idx int) error
	// TearDown tears down the client.
	TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error
}

// OnScheduleClientExtensions is an interface for on schedule client, e.g.
// the client is scheduled by the TiPocket test suite
type OnScheduleClientExtensions interface {
	Client
	// Invoke invokes a request to the database.
	// Mostly, the return Response should implement UnknownResponse interface
	Invoke(ctx context.Context, node cluster.ClientNode, r interface{}) UnknownResponse
	// NextRequest generates a request for latter Invoke.
	NextRequest() interface{}
	// DumpState the database state(also the model's state)
	DumpState(ctx context.Context) (interface{}, error)
}

// StandardClientExtensions is an interface for auto driver client, e.g.
// the client take over control from the TiPocket test suite
type StandardClientExtensions interface {
	Client
	// Start runs auto driver cases
	// this function will block Invoke trigger
	// if you want to schedule cases by yourself, use this function only
	Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error
}

// ClientCreator creates a client.
// The control will create one client for one node.
type ClientCreator interface {
	// Create creates the client.
	Create(node cluster.ClientNode) Client
}

// NoopClientCreator creates a noop client.
type NoopClientCreator struct {
}

// Create creates the client.
func (NoopClientCreator) Create(node cluster.Node) Client {
	return noopClient{}
}

// noopClient is a noop client
type noopClient struct {
}

func (c noopClient) ScheduledClientExtensions() OnScheduleClientExtensions {
	return nil
}

func (c noopClient) StandardClientExtensions() StandardClientExtensions {
	return c
}

// SetUp sets up the client.
func (noopClient) SetUp(ctx context.Context, _ []cluster.Node, _ []cluster.ClientNode, idx int) error {
	return nil
}

// TearDown tears down the client.
func (noopClient) TearDown(ctx context.Context, nodes []cluster.ClientNode, idx int) error {
	return nil
}

// Start runs auto driver cases
func (noopClient) Start(ctx context.Context, cfg interface{}, clientNodes []cluster.ClientNode) error {
	return nil
}

type noopResponse struct{}

func (noopResponse) IsUnknown() bool {
	return false
}
