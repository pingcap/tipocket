package types

import (
	"context"
	"fmt"
)

// Component is the identifier of Cluster
type Component string

const (
	// TiDB component identifier
	TiDB Component = "tidb"
	// TiKV component identifier
	TiKV Component = "tikv"
	// PD component identifier
	PD Component = "pd"
	// Pump Component identifier
	Pump Component = "pump"
	// Drainer Component identifier
	Drainer Component = "drainer"
	// Unknown component identifier
	Unknown Component = "unknown"
)

// Client provides useful methods about cluster
type Client struct {
	Namespace    string
	PDMemberFunc func(ns string) (string, []string, error)
}

func (c *Client) PDMember() (string, []string, error) {
	return c.PDMemberFunc(c.Namespace)
}

// Node is the cluster endpoint in K8s, it's maybe podIP:port or CLUSTER-IP:port
type Node struct {
	Namespace string    // Cluster k8s' namespace
	Component Component // Node component type
	PodName   string    // Pod's name
	IP        string
	Port      int32
	*Client   `json:"-"`
}

// ClientNode is TiDB's exposed endpoint, can be a nodeport, or downgrade cluster ip
type ClientNode struct {
	Namespace string // Cluster k8s' namespace
	Component Component
	IP        string
	Port      int32
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
