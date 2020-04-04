package types

// Cluster interface
type Cluster interface {
	Apply() error
	Delete() error
	GetNodes() ([]Node, error)
	GetClientNodes() ([]ClientNode, error)
}
