package types

// Cluster interface
type Cluster interface {
	Namespace() string
	Apply() error
	Delete() error
	GetNodes() ([]Node, error)
	GetClientNodes() ([]ClientNode, error)
}
