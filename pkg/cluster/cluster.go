package cluster

import (
	"context"
	"fmt"
)

// Node is the cluster endpoint in K8s, it's maybe podIP:port or CLUSTER-IP:port
type Node struct {
	// Cluster k8s' namespace
	Namespace string
	// Pod's name
	PodName string

	IP   string
	Port int32
}

// ClientNode is TiDB's exposed endpoint, can be a nodeport, or downgrade cluster ip
type ClientNode struct {
	// Cluster k8s' namespace
	Namespace string

	IP   string
	Port int32
}

// Provisioner provides a collection of APIs to deploy/destroy a cluster
type Provisioner interface {
	// SetUp sets up cluster, returns err or all nodes info
	SetUp(ctx context.Context, spec interface{}) ([]Node, []ClientNode, error)
	// TearDown tears down the cluster
	TearDown() error
}

func (node Node) String() string {
	return fmt.Sprintf("%s %s:%d", node.PodName, node.IP, node.Port)
}

func (node ClientNode) String() string {
	return fmt.Sprintf("%s:%d", node.IP, node.Port)
}
