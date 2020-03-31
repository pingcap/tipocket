package operation

import clusterTypes "github.com/pingcap/tipocket/pkg/cluster/types"

type Cluster interface {
	Namespace() string
	Apply() error
	Delete() error
	GetNodes() ([]clusterTypes.Node, error)
	GetClientNodes() ([]clusterTypes.ClientNode, error)
}
