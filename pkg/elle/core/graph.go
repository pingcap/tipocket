package core

import "github.com/mohae/deepcopy"

// Edge is a intermediate representation of edge on DirectedGraph
type Edge struct {
	From  Vertex
	To    Vertex
	Value Rel // raw value of edge
}
type Vertex struct {
	Value interface{}
}

type DirectedGraph struct {
	Outs map[Vertex]map[Vertex]Rel
	Ins  map[Vertex][]Vertex
}

// Vertices returns the set of all vertices in the node
func (g *DirectedGraph) Vertices() []Vertex {
	panic("impl me")
}

// In returns inbound edges to v in graph g
func (g *DirectedGraph) In(v Vertex) []Vertex {
	panic("impl me")
}

// Out returns outbound edges from v in graph g
func (g *DirectedGraph) Out(v Vertex) []Vertex {
	panic("impl me")
}

// Edges returns the edge between two vertices
func (g *DirectedGraph) Edges(a, b Vertex) []Edge {
	panic("impl me")
}

// Link links two vertices relationship
func (g *DirectedGraph) Link(v Vertex, succ []Vertex, rel Rel) {
	panic("impl me")
}

// LinkToAll links x to all ys
func (g *DirectedGraph) LinkToAll(x Vertex, ys []Vertex, rel Rel) {
	panic("impl me")
}

// LinkAllTo links all xs to y
func (g *DirectedGraph) LinkAllTo(xs []Vertex, y Vertex) {
	panic("impl me")
}

func (g *DirectedGraph) UnLink(a, b Vertex) {
	panic("impl me")
}

// Fork implements `forked` semantics on DirectedGraph
func (g *DirectedGraph) Fork() *DirectedGraph {
	return deepcopy.Copy(g).(*DirectedGraph)
}

// keepEdgeValues transforms a graph by applying a function (f edge-value) to each edge in the
//  graph. Where the function returns `nil`, removes that edge altogether
func (g *DirectedGraph) keepEdgeValues(f func(interface{}) interface{}) *DirectedGraph {
	panic("impl me")
}

// RemoveRelationship removes the given relationship from it
func (g *DirectedGraph) RemoveRelationship(rel Rel) *DirectedGraph {
	panic("impl me")
}

// ProjectRelationship filters a graph to just those edges with the given relationship
func (g *DirectedGraph) ProjectRelationship(rel Rel) *DirectedGraph {
	panic("impl me")
}

// FilterRelationships filters a graph g to just those edges which intersect with the given set of
//  relationships
func (g *DirectedGraph) FilterRelationships(rels []Rel) *DirectedGraph {
	panic("impl me")
}

// SCC indexes all vertices of a strongly connected component
type SCC struct {
	Vertices []Vertex
}

// TODO
type Transition struct{}

// TODO
type Predicate struct{}

// SCCs receives a direct graph and finds all strongly connected components
func SCCs(graph DirectedGraph) []SCC {
	panic("impl me")
}

// FindCycle receives a graph and a scc, finds a short cycle in that component
func FindCycle(graph DirectedGraph, scc SCC) Circle {
	panic("impl me")
}

// FindCycleStartingWith ...
func FindCycleStartingWith(initialGraph, remainingGraph DirectedGraph, scc SCC) Circle {
	panic("impl me")
}

// FindCycleWith ...
func FindCycleWith(transition Transition, pred Predicate, graph DirectedGraph, scc SCC) Circle {
	panic("impl me")
}
