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

	for key := range g.Outs {
		vertices = append(vertices, key)
	}
	return vertices
}

// In returns inbound vertices to v in graph g
func (g *DirectedGraph) In(v Vertex) []Vertex {
	_, ok := g.Ins[v]
	if !ok {
		return []Vertex{}
	} else {
		return g.Ins[v]
	}
}

// Out returns outbound vertices from v in graph g
func (g *DirectedGraph) Out(v Vertex) []Vertex {
	_, ok := g.Outs[v]
	if !ok {
		return []Vertex{}
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
		return []Edge{}
	}

	_, ok := g.Outs[a][b]
	if !ok {
		return []Edge{}
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

	haveRel := false
	for _, item := range g.Outs[v][succ] {
		if item == rel {
			haveRel = true
			break
		}
	}
	if haveRel == false {
		g.Outs[v][succ] = append(g.Outs[v][succ], rel)
	}

	haveV := false
	for _, item := range g.Ins[succ] {
		if item == v {
			haveV = true
			break
		}
	}
	if haveV == false {
		g.Ins[succ] = append(g.Ins[succ], v)
	}
}

// LinkToAll links x to all ys
func (g *DirectedGraph) LinkToAll(x Vertex, ys []Vertex, rel Rel) {
	for _, y := range ys {
		g.Link(x, y, rel)
	}
}

// LinkAllTo links all xs to y
func (g *DirectedGraph) LinkAllTo(xs []Vertex, y Vertex, rel Rel) {
	for _, x := range xs {
		g.Link(x, y, rel)
	}
}

func (g *DirectedGraph) UnLink(a, b Vertex) {
	delete(g.Outs[a], b)

	for id , vertex := range g.Ins[b] {
		if vertex == a {
			g.Ins[b] = append(g.Ins[b][:id], g.Ins[b][id+1:]...)
			break
		}
	}
}

// Fork implements `forked` semantics on DirectedGraph
func (g *DirectedGraph) Fork() *DirectedGraph {
	return deepcopy.Copy(g).(*DirectedGraph)
}
// IntersectionRel returns the union of res1 and res2
func IntersectionRel(rels1 []Rel,rels2 []Rel) []Rel{
	rel := []Rel{}
	for _, rel1 := range rels1 {
		both := false
		for _, rel2 := range rels2{
			if rel1 == rel2 {
				both = true
				break
			}
		}
		if both == true {
			rel = append(rel, rel1)
		}
	}
	return rel
}

// ProjectRelationship filters a graph to just those edges with the given relationship
func (g *DirectedGraph) ProjectRelationship(rel Rel) *DirectedGraph {
	fg := g.Fork()

	vertices := fg.Vertices()
	for _, x := range vertices {
		_, ok := g.Outs[x]
		if ok == true {
			for y, _:= range g.Outs[x] {
				rels := IntersectionRel(g.Outs[x][y], []Rel{rel})
				fg.UnLink(x, y)
				for _, item := range rels {
					fg.Link(x, y, item)
				}
			}
		}
	}

	return fg
}

// FilterRelationships filters a graph g to just those edges which intersect with the given set of
//  relationships
func (g *DirectedGraph) FilterRelationships(rels []Rel) *DirectedGraph {
	fg := g.Fork()

	vertices := fg.Vertices()
	for _, x := range vertices {
		_, ok := g.Outs[x]
		if ok == true {
			for y, _:= range g.Outs[x] {
				rels := IntersectionRel(g.Outs[x][y], rels)
				fg.UnLink(x, y)
				for _, item := range rels {
					fg.Link(x, y, item)
				}
			}
		}
	}

	return fg
}

// BfsOut searches from a vertices set, returns all vertices searchable
// search to downstream vertices from `out` edges
func (g *DirectedGraph) BfsOut(initV []Vertex) []Vertex {
	queue := initV
	V := []Vertex{}
	haveVisited := make(map[Vertex]bool)
	for _, vertex := range initV {
		haveVisited[vertex] = true
		V = append(V, vertex)
	}

	for len(queue) > 0 {
		cur := queue[0]
		if haveVisited[cur] == false {
			haveVisited[cur] = true
			V = append(V, cur)
		}
		queue = queue[1:]

		for next, _ := range g.Outs[cur] {
			if haveVisited[next] == true {
				continue
			}
			queue = append(queue, next)
		}
	}
	return V
}

// BfsIn searches from a vertices set, returns all vertices searchable
// search to upstream vertices from `in` edges
func (g *DirectedGraph) BfsIn(initV []Vertex) []Vertex {
	queue := initV
	V := []Vertex{}
	haveVisited := make(map[Vertex]bool)
	for _, vertex := range initV {
		haveVisited[vertex] = true
		V = append(V, vertex)
	}

	for len(queue) > 0 {
		cur := queue[0]
		if haveVisited[cur] == false {
			haveVisited[cur] = true
			V = append(V, cur)
		}
		queue = queue[1:]

		for _, next := range g.Ins[cur] {
			if haveVisited[next] == true {
				continue
			}
			queue = append(queue, next)
		}
	}
	return V
}

// Bfs searches from a vertices set, returns all vertices searchable
// out = true means search to downstream vertices from `out` edges
// out = false means search to upstream vertices from `in` edges
func (g *DirectedGraph) Bfs(initV []Vertex, out bool) []Vertex {
	if out == true {
		return g.BfsOut(initV)
	} else {
		return g.BfsIn(initV)
	}
}

// SCC indexes all vertices of a strongly connected component
type SCC struct {
	Vertices []Vertex
}

// StronglyConnectedComponents finds all strongly connected components
func (g *DirectedGraph) StronglyConnectedComponents() []SCC {
	dfn := make(map[Vertex]int)
	low := make(map[Vertex]int)
	cnt := 0
	belong := make(map[Vertex]int)
	in := make(map[Vertex]bool)
	tag := make(map[Vertex]map[Vertex]bool)
	s2 := []Vertex{}
	scc := []SCC{}
	vertices := g.Vertices()

	for _, v := range vertices {
		if dfn[v] == 0 {
			s1 := []Vertex{v}
			index := 0
			for len(s1) > 0 {
				x := s1[len(s1)-1]
				s1 = s1[:len(s1)-1]
				if dfn[x] == 0 {
					index = index + 1
					dfn[x] = index
					low[x] = index
					s2 = append(s2, x)
					in[x] = true;
				}
				finish := true
				for next, _ := range g.Outs[x] {
					if dfn[next] == 0 {
						s1 = append(s1, x)
						s1 = append(s1, next)
						_, ok := tag[x]
						if !ok {
							tag[x] = make(map[Vertex]bool)
						}
						tag[x][next] = true
						finish = false
						break
					} else {
						if in[next] == true {
							if low[x] > dfn[next] {
								low[x] = dfn[next]
							}
						}
					}
				}
				if finish == true {
					for next, _ := range g.Outs[x] {
						_, ok := tag[x]
						if !ok {
							tag[x] = make(map[Vertex]bool)
						}
						if tag[x][next] == true {
							if low[x] > low[next] {
								low[x] = low[next]
							}
						}
					}
					if low[x] == dfn[x] {
						var t Vertex
						cnt = cnt + 1
						for {
							t = s2[len(s2)-1]
							s2 = s2[:len(s2)-1]
							in[t] = false
							belong[t] = cnt;
							if t == x {
								break
							}
						}
					}
				}
			}
		}
	}
	for i := 1; i <= cnt; i++ {
		ts := SCC{
			Vertices: []Vertex{},
		}
		for _, v := range vertices {
			if belong[v] == i {
				ts.Vertices = append(ts.Vertices, v)
			}
		}
		scc = append(scc, ts)
	}
	return scc
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
