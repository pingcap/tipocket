package cluster

import (
	"context"
	"encoding/json"
)

// Node is the service access point in K8s, it's maybe podIP:port or CLUSTER-IP:port
type Node struct {
	IP   string
	Port string
}

// Provisioner provides a collection of APIs to deploy/destroy a cluster
type Provisioner interface {
	// SetUp sets up cluster, returns err or all nodes info
	SetUp(ctx context.Context, spec json.RawMessage) (error, []Node)
	// TearDown tears down the cluster
	TearDown() error
}
