package core

// Edge is a intermediate representation of edge on DirectedGraph
type Edge struct {
	From  Vertex
	To    Vertex
	Value Rel // raw value of edge
}

type Vertex struct {
	Value interface{}
}

type Rels []Rel

func (rs Rels) exist(rel Rel) bool {
	for _, r := range rs {
		if r == rel {
			return true
		}
	}
	return false
}

type DirectedGraph struct {
	Outs map[Vertex]map[Vertex][]Rel
	Ins  map[Vertex][]Vertex
}

// Vertices returns the set of all vertices in graph
func (g *DirectedGraph) Vertices() []Vertex {
	var vertices []Vertex

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

	var vertices []Vertex

	for key := range g.Outs[v] {
		vertices = append(vertices, key)
	}
	return vertices
}

// Edges returns the edge between two vertices
func (g *DirectedGraph) Edges(a, b Vertex) []Edge {
	_, oka := g.Outs[a]
	if !oka {
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
	_, ok = g.Outs[succ]
	if !ok {
		g.Outs[succ] = make(map[Vertex][]Rel)
	}

	_, ok = g.Ins[v]
	if !ok {
		g.Ins[v] = []Vertex{}
	}
	_, ok = g.Ins[succ]
	if !ok {
		g.Ins[succ] = []Vertex{}
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

	//if _, ok := g.Outs[v][succ]; ok == false {
	//  g.Ins[succ] = append(g.Ins[succ], v)
	//}
	g.Ins[succ] = append(g.Ins[succ], v)
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

// LinkAllToAll links all xs to ys
func (g *DirectedGraph) LinkAllToAll(xs []Vertex, ys []Vertex, rel Rel) {
	for _, x := range xs {
		for _, y := range ys {
			g.Link(x, y, rel)
		}
	}
}

func (g *DirectedGraph) UnLink(a, b Vertex) {
	delete(g.Outs[a], b)

	for id, vertex := range g.Ins[b] {
		if vertex == a {
			g.Ins[b] = append(g.Ins[b][:id], g.Ins[b][id+1:]...)
			break
		}
	}
}

// Fork implements `forked` semantics on DirectedGraph
func (g *DirectedGraph) Fork() *DirectedGraph {
	d := NewDirectedGraph()
	for _, a := range g.Vertices() {
		for _, b := range g.Out(a) {
			for _, e := range g.Edges(a, b) {
				d.Link(a, b, e.Value)
			}
		}
	}
	return d
}

// IntersectionRel returns the intersection of res1 and res2
func IntersectionRel(rels1 []Rel, rels2 []Rel) []Rel {
	var rel []Rel
	for _, rel1 := range rels1 {
		for _, rel2 := range rels2 {
			if rel1 == rel2 {
				rel = append(rel, rel1)
			}
		}
	}
	return rel
}

// ProjectRelationship filters a graph to just those edges with the given relationship
func (g *DirectedGraph) ProjectRelationship(rel Rel) *DirectedGraph {
	return g.FilterRelationships([]Rel{rel})
}

// FilterRelationships filters a graph g to just those edges which intersect with the given set of
//  relationships
func (g *DirectedGraph) FilterRelationships(rels []Rel) *DirectedGraph {
	fg := g.Fork()

	vertices := fg.Vertices()
	for _, x := range vertices {
		_, ok := g.Outs[x]
		if ok == true {
			for y, _ := range g.Outs[x] {
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
	vertices := []Vertex{}
	haveVisited := make(map[Vertex]bool)
	for _, vertex := range initV {
		haveVisited[vertex] = true
		vertices = append(vertices, vertex)
	}

	for len(queue) > 0 {
		cur := queue[0]
		if haveVisited[cur] == false {
			haveVisited[cur] = true
			vertices = append(vertices, cur)
		}
		queue = queue[1:]

		for next, _ := range g.Outs[cur] {
			if haveVisited[next] == true {
				continue
			}
			queue = append(queue, next)
		}
	}
	return vertices
}

// BfsIn searches from a vertices set, returns all vertices searchable
// search to upstream vertices from `in` edges
func (g *DirectedGraph) BfsIn(initV []Vertex) []Vertex {
	queue := initV
	vertices := []Vertex{}
	haveVisited := make(map[Vertex]bool)
	for _, vertex := range initV {
		haveVisited[vertex] = true
		vertices = append(vertices, vertex)
	}

	for len(queue) > 0 {
		cur := queue[0]
		if haveVisited[cur] == false {
			haveVisited[cur] = true
			vertices = append(vertices, cur)
		}
		queue = queue[1:]

		for _, next := range g.Ins[cur] {
			if haveVisited[next] == true {
				continue
			}
			queue = append(queue, next)
		}
	}
	return vertices
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

// StronglyConnectedComponents finds all strongly connected components, greater than 1 element
func (g *DirectedGraph) StronglyConnectedComponents() []SCC {
	dfn := make(map[Vertex]int)
	low := make(map[Vertex]int)
	cnt := 0
	belong := make(map[Vertex]int)
	in := make(map[Vertex]bool)
	tag := make(map[Vertex]map[Vertex]bool)
	var s2 []Vertex
	vertices := g.Vertices()
	index := 0
	for _, v := range vertices {
		if dfn[v] == 0 {
			s1 := []Vertex{v}
			for len(s1) > 0 {
				x := s1[len(s1)-1]
				s1 = s1[:len(s1)-1]
				if dfn[x] == 0 {
					index = index + 1
					dfn[x] = index
					low[x] = index
					s2 = append(s2, x)
					in[x] = true
				}
				finish := true
				for next := range g.Outs[x] {
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
					for next := range g.Outs[x] {
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
							belong[t] = cnt
							if t == x {
								break
							}
						}
					}
				}
			}
		}
	}

	scc := make([]SCC, cnt)
	for _, v := range vertices {
		id := belong[v]
		scc[id-1].Vertices = append(scc[id-1].Vertices, v)
	}

	var sccs []SCC
	for _, sc := range scc {
		if len(sc.Vertices) >= 2 {
			sccs = append(sccs, sc)
		}
	}
	return sccs
}

// NewDirectedGraph returns a empty DirectedGraph
func NewDirectedGraph() *DirectedGraph {
	var g DirectedGraph
	g.Ins = make(map[Vertex][]Vertex)
	g.Outs = make(map[Vertex]map[Vertex][]Rel)
	return &g
}

// MapVertices takes a function of vertices, returns graph with all
//  vertices mapped via f
func (g *DirectedGraph) MapVertices(f func(interface{}) interface{}) *DirectedGraph {
	fg := NewDirectedGraph()
	vertices := g.Vertices()
	for _, x := range vertices {
		_, ok := g.Outs[x]
		if ok == true {
			for y, _ := range g.Outs[x] {
				rels := g.Outs[x][y]
				for _, item := range rels {
					fg.Link(Vertex{f(x.Value)}, Vertex{f(y.Value)}, item)
				}
			}
		}
	}

	return fg
}

// RenumberGraph takes a Graph and rewrites each vertex to a unique integer, returning the
//  rewritten Graph, and a vector of the original vertices for reconstruction.
// That means if we apply MapVertices to dg with remap, it can recover to g again
func (g *DirectedGraph) RenumberGraph() (*DirectedGraph, func(interface{}) interface{}) {
	dg := NewDirectedGraph()
	ref := make(map[Vertex]int)
	numberToVertices := make(map[interface{}]interface{})
	vertices := g.Vertices()

	for id, v := range vertices {
		ref[v] = id
		numberToVertices[id] = v.Value
	}

	for _, x := range vertices {
		for y, rels := range g.Outs[x] {
			for _, rel := range rels {
				dg.Link(Vertex{ref[x]}, Vertex{ref[y]}, rel)
			}
		}
	}

	return dg, func(number interface{}) interface{} {
		return numberToVertices[number]
	}
}

// MapToDirectedGraph turns a sequence of [node, successors] map into a directed graph
func MapToDirectedGraph(m map[Vertex][]Vertex) *DirectedGraph {
	dg := NewDirectedGraph()
	for x, vertices := range m {
		for _, y := range vertices {
			dg.Link(x, y, "")
		}
	}
	return dg
}

// DigraphUnion takes the union of n graphs, merging edges with union
func DigraphUnion(graphs ...*DirectedGraph) *DirectedGraph {
	dg := NewDirectedGraph()
	for _, g := range graphs {
		vertices := g.Vertices()
		for _, x := range vertices {
			for y, rels := range g.Outs[x] {
				for _, rel := range rels {
					dg.Link(x, y, rel)
				}
			}
		}
	}
	return dg
}

type CyclePredicate func(trace []CycleTrace) bool

// FindCycle receives a graph and a scc, finds a short cycle in that component
// TODO: find the shortest cycle
func FindCycle(graph *DirectedGraph, scc SCC) []Vertex {
	if len(scc.Vertices) == 1 {
		return []Vertex{}
	}
	length := len(scc.Vertices) + 1
	sccSet := toSet(scc.Vertices)

	var destCycle []CycleTrace
	for _, start := range scc.Vertices {
		bfs := NewBFSPath(graph, start, sccSet)
		for _, next := range graph.Out(start) {
			if _, e := sccSet[next]; !e {
				continue
			}
			if bfs.HasPathFrom(next) && (bfs.DistFrom(next)+1) < length {
				length = bfs.DistFrom(next) + 1
				destCycle = append([]CycleTrace{{from: start, Rels: getRelsFromEdges(graph.Edges(start, next))}}, bfs.PathFrom(next)...)
			}
		}
	}
	return getVerticesFromTracePath(destCycle)
}

// FindCycleStartingWith ...
// TODO: find the shortest cycle
func FindCycleStartingWith(graph *DirectedGraph, rel Rel, scc SCC) []Vertex {
	if len(scc.Vertices) == 1 {
		return []Vertex{}
	}
	length := len(scc.Vertices) + 1
	sccSet := toSet(scc.Vertices)

	var destCycle []CycleTrace
	for _, start := range scc.Vertices {
		bfs := NewBFSPath(graph, start, sccSet)
		out, ok := graph.Outs[start]
		if !ok {
			continue
		}
		for next, rs := range out {
			if _, e := sccSet[next]; !e {
				continue
			}
			if !Rels(rs).exist(rel) {
				continue
			}
			if bfs.HasPathFrom(next) && (bfs.DistFrom(next)+1) < length {
				length = bfs.DistFrom(next) + 1
				destCycle = append([]CycleTrace{{from: start, Rels: getRelsFromEdges(graph.Edges(start, next))}}, bfs.PathFrom(next)...)
			}
		}
	}
	return getVerticesFromTracePath(destCycle)
}

// FindCycleWith ...
func FindCycleWith(graph *DirectedGraph, scc SCC, isWith CyclePredicate) []Vertex {
	if len(scc.Vertices) == 1 {
		return []Vertex{}
	}
	length := len(scc.Vertices) + 1
	sccSet := toSet(scc.Vertices)

	var destCycle []CycleTrace
	for _, start := range scc.Vertices {
		bfs := NewBFSPath(graph, start, sccSet)
		out, ok := graph.Outs[start]
		if !ok {
			continue
		}
		for next := range out {
			if _, e := sccSet[next]; !e {
				continue
			}
			if bfs.HasPathFrom(next) && (bfs.DistFrom(next)+1) < length {
				cycle := append([]CycleTrace{{from: start, Rels: getRelsFromEdges(graph.Edges(start, next))}}, bfs.PathFrom(next)...)
				// cycle: t1(rels of t1 and t2) -> t2(rels of t2 and t1) -> t1(rels of t1 and t2),
				// so we need remove the last element when we invoke isWith
				if isWith(cycle[:len(cycle)-1]) {
					destCycle = cycle
					length = bfs.DistFrom(next) + 1
				}
			}
		}
	}
	return getVerticesFromTracePath(destCycle)
}
