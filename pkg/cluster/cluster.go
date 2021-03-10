package cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/pingcap/tipocket/pkg/test-infra/fixture"
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
	// TiCDC Component identifier
	TiCDC Component = "ticdc"
	// DM Component identifier
	DM Component = "dm"
	// Monitor Component identifier
	Monitor Component = "monitor"
	// TiFlash Component identifier
	TiFlash Component = "tiflash"
	// MySQL Component identifier
	MySQL Component = "mysql"
	// Unknown component identifier
	Unknown Component = "unknown"
)

// Client provides useful methods about cluster
type Client struct {
	Namespace    string
	ClusterName  string
	PDMemberFunc func(ns, name string) (string, []string, error)
}

// PDMember ...
func (c *Client) PDMember() (string, []string, error) {
	return c.PDMemberFunc(c.Namespace, c.ClusterName)
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

// Address returns the endpoint address of node
func (node Node) Address() string {
	return fmt.Sprintf("%s:%d", node.IP, node.Port)
}

// String ...
func (node Node) String() string {
	sb := new(strings.Builder)
	fmt.Fprintf(sb, "node[comp=%s,ip=%s:%d", node.Component, node.IP, node.Port)
	if node.Namespace != "" {
		fmt.Fprintf(sb, ",ns=%s", node.Namespace)
	}
	if node.PodName != "" {
		fmt.Fprintf(sb, ",pod=%s", node.PodName)
	}
	fmt.Fprint(sb, "]")
	return sb.String()
}

// ClientNode is TiDB's exposed endpoint, can be a nodeport, or downgrade cluster ip
type ClientNode struct {
	Namespace   string // Cluster k8s' namespace
	ClusterName string // Cluster name, use to differentiate different TiDB clusters running on same namespace
	Component   Component
	IP          string
	Port        int32
}

// Address returns the endpoint address of clientNode
func (clientNode ClientNode) Address() string {
	return fmt.Sprintf("%s:%d", clientNode.IP, clientNode.Port)
}

// String ...
func (clientNode ClientNode) String() string {
	sb := new(strings.Builder)
	fmt.Fprintf(sb, "client_node[comp=%s,ip=%s:%d", clientNode.Component, clientNode.IP, clientNode.Port)
	if clientNode.Namespace != "" {
		fmt.Fprintf(sb, ",ns=%s", clientNode.Namespace)
	}
	if clientNode.ClusterName != "" {
		fmt.Fprintf(sb, ",cluster=%s", clientNode.ClusterName)
	}
	fmt.Fprint(sb, "]")
	return sb.String()
}

// Cluster interface
type Cluster interface {
	Apply() error
	Delete() error
	GetNodes() ([]Node, error)
	GetClientNodes() ([]ClientNode, error)
}

// Specs is a cluster specification
type Specs struct {
	Cluster     Cluster
	NemesisGens []string
	Namespace   string
}

// Provider provides a collection of APIs to deploy/destroy a cluster
type Provider interface {
	// SetUp sets up cluster, returns err or all nodes info
	SetUp(ctx context.Context, spec Specs) ([]Node, []ClientNode, error)
	// TearDown tears down the cluster
	TearDown(ctx context.Context, spec Specs) error
}

// NewDefaultClusterProvider ...
func NewDefaultClusterProvider() Provider {
	if len(fixture.Context.TiDBClusterConfig.TiDBAddr) != 0 ||
		len(fixture.Context.TiDBClusterConfig.TiKVAddr) != 0 ||
		len(fixture.Context.TiDBClusterConfig.PDAddr) != 0 {
		return NewLocalClusterProvisioner(fixture.Context.TiDBClusterConfig.TiDBAddr,
			fixture.Context.TiDBClusterConfig.PDAddr,
			fixture.Context.TiDBClusterConfig.TiKVAddr)
	}
	return NewK8sClusterProvider()
}
