package cluster

import (
	"context"
	"log"
	"strconv"
	"strings"

	"github.com/pingcap/tipocket/pkg/cluster/types"
)

type nodeComponent struct {
	ip        string
	port      int32
	component types.Component
}

// NewLocalClusterProvisioner reuses a local cluster
func NewLocalClusterProvisioner(dbs, pds, kvs []string) types.Provisioner {
	return &LocalClusterProvisioner{
		DBs: dbs,
		PDs: pds,
		KVs: kvs,
	}
}

type LocalClusterProvisioner struct {
	DBs []string
	PDs []string
	KVs []string
}

// SetUp fills nodes and clientNodes
func (l *LocalClusterProvisioner) SetUp(ctx context.Context, _ interface{}) ([]types.Node, []types.ClientNode, error) {
	var nodes []types.Node
	var clientNode []types.ClientNode
	var nodeComponents = buildNodeComponent(l.DBs, types.TiDB)
	nodeComponents = append(nodeComponents, buildNodeComponent(l.KVs, types.TiKV)...)
	nodeComponents = append(nodeComponents, buildNodeComponent(l.PDs, types.PD)...)

	for _, node := range nodeComponents {
		nodes = append(nodes, types.Node{
			Namespace: "",
			Component: node.component,
			Version:   "",
			PodName:   "",
			IP:        node.ip,
			Port:      node.port,
			Client:    nil,
		})
	}

	for _, node := range nodeComponents {
		if node.component != types.TiDB {
			continue
		}
		clientNode = append(clientNode, types.ClientNode{
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
func (l *LocalClusterProvisioner) TearDown(ctx context.Context, spec interface{}) error {
	return nil
}

func buildNodeComponent(nodes []string, component types.Component) []nodeComponent {
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
