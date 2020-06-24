package cluster

import (
	"context"
	"log"
	"strconv"
	"strings"
)

type nodeComponent struct {
	ip        string
	port      int32
	component Component
}

// NewLocalClusterProvisioner reuses a local cluster
func NewLocalClusterProvisioner(dbs, pds, kvs []string) Provider {
	return &LocalClusterProvider{
		DBs: dbs,
		PDs: pds,
		KVs: kvs,
	}
}

// LocalClusterProvider ...
type LocalClusterProvider struct {
	DBs []string
	PDs []string
	KVs []string
}

// SetUp fills nodes and clientNodes
func (l *LocalClusterProvider) SetUp(ctx context.Context, _ Specs) ([]Node, []ClientNode, error) {
	var nodes []Node
	var clientNode []ClientNode
	var nodeComponents = buildNodeComponent(l.DBs, TiDB)
	nodeComponents = append(nodeComponents, buildNodeComponent(l.KVs, TiKV)...)
	nodeComponents = append(nodeComponents, buildNodeComponent(l.PDs, PD)...)

	for _, node := range nodeComponents {
		nodes = append(nodes, Node{
			Namespace: "",
			Component: node.component,
			PodName:   "",
			IP:        node.ip,
			Port:      node.port,
			Client:    nil,
		})
	}

	for _, node := range nodeComponents {
		if node.component != TiDB {
			continue
		}
		clientNode = append(clientNode, ClientNode{
			Namespace:   "",
			ClusterName: "",
			Component:   node.component,
			IP:          node.ip,
			Port:        node.port,
		})
	}
	return nodes, clientNode, nil
}

// TearDown does nothing here
func (l *LocalClusterProvider) TearDown(ctx context.Context, _ Specs) error {
	return nil
}

func buildNodeComponent(nodes []string, component Component) []nodeComponent {
	var nodeComponents []nodeComponent
	for _, node := range nodes {
		addr := strings.Split(node, ":")
		if len(addr) != 2 {
			log.Fatalf("expect format ip:port, got %s", addr)
		}
		ip := addr[0]
		port, err := strconv.Atoi(addr[1])
		if err != nil {
			log.Fatalf("illegal port %s", addr[1])
		}
		nodeComponents = append(nodeComponents, nodeComponent{ip: ip, port: int32(port), component: component})
	}
	return nodeComponents
}
