package cluster

import "encoding/json"

// Node is the service access point in K8s, it's maybe podIP:port or CLUSTER-IP:port
type Node struct {
	IP   string
	Port string
}

// Provisioner provides APIs for deploy/destroy a cluster
type Provisioner interface {
	Deploy(spec json.RawMessage) (error, []Node)
	Destroy() error
}
