package cluster

import (
	"context"
)

// Node is the service endpoint in K8s, it's maybe podIP:port or CLUSTER-IP:port
type Node struct {
	// Cluster k8s' namespace
	Namespace string
	// Pod's name
	PodName string

	IP   string
	Port string
}

// Provisioner provides a collection of APIs to deploy/destroy a cluster
type Provisioner interface {
	// SetUp sets up cluster, returns err or all nodes info
	SetUp(ctx context.Context, spec interface{}) (error, []Node)
	// TearDown tears down the cluster
	TearDown() error
}
