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
	Outs map[Vertex]map[Vertex][]Rel
	Ins  map[Vertex][]Vertex
}

// Vertices returns the set of all vertices in graph
func (g *DirectedGraph) Vertices() []Vertex {
	vertices := []Vertex{}
	have_in := make(map[Vertex]bool)

	for key := range g.Ins {
		vertices = append(vertices, key)
		have_in[key] = true
	}

	for key := range g.Outs {
		_, ok := have_in[key]
		if !ok {
			vertices = append(vertices, key)
		}
	}
	return vertices
}

// In returns inbound vertices to v in graph g
func (g *DirectedGraph) In(v Vertex) []Vertex {
	_, ok := g.Ins[v]
	if !ok {
		return nil
	} else {
		return g.Ins[v]
	}
}

// Out returns outbound vertices from v in graph g
func (g *DirectedGraph) Out(v Vertex) []Vertex {
	_, ok := g.Outs[v]
	if !ok {
		return nil
	}

	vertices := []Vertex{}

	for key := range g.Outs[v] {
		vertices = append(vertices, key)
	}
	return vertices
}

// Edges returns the edge between two vertices
func (g *DirectedGraph) Edges(a, b Vertex) []Edge {
	_, oka := g.Outs[a]
	_, okb := g.Ins[b]
	if !oka || !okb {
		return nil
	}

	_, ok := g.Outs[a][b]
	if !ok {
		return nil
	}

	edges := []Edge{}
	for _, rel := range g.Outs[a][b] {
		edges = append(edges, Edge{
			From:  a,
			To:    b,
			Value: rel,
		})
	}
	return edges
}

// Link links two vertices relationship
func (g *DirectedGraph) Link(v Vertex, succ Vertex, rel Rel) {

	_, ok := g.Outs[v]
	if !ok {
		g.Outs[v] = make(map[Vertex][]Rel)
	}

	g.Outs[v][succ] = append(g.Outs[v][succ], rel)
	g.Ins[succ] = append(g.Ins[succ], v)
}

// LinkToAll links x to all ys
func (g *DirectedGraph) LinkToAll(x Vertex, ys []Vertex, rel Rel) {
	for _, y := range ys {

		_, ok := g.Outs[x]
		if !ok {
			g.Outs[x] = make(map[Vertex][]Rel)
		}

		g.Outs[x][y] = append(g.Outs[x][y], rel)
		g.Ins[y] = append(g.Ins[y], x)
	}
}

// LinkAllTo links all xs to y
func (g *DirectedGraph) LinkAllTo(xs []Vertex, y Vertex, rel Rel) {
	for _, x := range xs {

		_, ok := g.Outs[x]
		if !ok {
			g.Outs[x] = make(map[Vertex][]Rel)
		}

		g.Outs[x][y] = append(g.Outs[x][y], rel)
		g.Ins[y] = append(g.Ins[y], x)
	}
}

func (g *DirectedGraph) UnLink(a, b Vertex) {
	delete(g.Outs[a], b)

	have_a := false
	for {
		for id , vertex := range g.Ins[b] {
			if vertex == a{
				have_a = true
				g.Ins[b] = append(g.Ins[b][:id], g.Ins[b][id+1:]...)
				break
			}
		}
		if have_a == false {
			break
		}
	}
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

// Bfs searches from a vertices set, returns all vertices searchable
// out = true means search to downstream vertices from `out` edges
// out = false means search to upstream vertices from `in` edges
func (g *DirectedGraph) Bfs(initV []Vertex, out bool) []Vertex {
	panic("impl me")
}

// SCC indexes all vertices of a strongly connected component
type SCC struct {
	Vertices []Vertex
}

// StronglyConnectedComponents finds all strongly connected components
func (g *DirectedGraph) StronglyConnectedComponents() []SCC {
	panic("impl me")
}

// MapVertices takes a function of vertices, returns graph with all
//  vertices mapped via f
func (g *DirectedGraph) MapVertices(f func(interface{}) interface{}) *DirectedGraph {
	panic("impl me")
}

// RenumberGraph takes a Graph and rewrites each vertex to a unique integer, returning the
//  rewritten Graph, and a vector of the original vertexes for reconstruction.
// That means if we apply MapVertices to dg with remap, it can recover to g again
func (g *DirectedGraph) RenumberGraph() (dg *DirectedGraph, remap func(interface{}) interface{}) {
	panic("impl me")
}

// MapToDirectedGraph turns a sequence of [node, successors] map into a directed graph
func MapToDirectedGraph(m map[Vertex][]Vertex) *DirectedGraph {
	panic("impl me")
}

// DigraphUion takes the union of n graphs, merging edges with union
func DigraphUion(graphs ...DirectedGraph) *DirectedGraph {
	panic("impl me")
}

// TODO
type Transition struct{}

// TODO
type Predicate struct{}

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
