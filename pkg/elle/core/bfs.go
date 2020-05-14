package core

// BFSPath ...
type BFSPath struct {
	g      *DirectedGraph
	marked map[Vertex]struct{}
	edgeTo map[Vertex]Vertex
	distTo map[Vertex]int
}

// NewBFSPath ...
func NewBFSPath(graph *DirectedGraph, start Vertex, sccSet map[Vertex]struct{}) *BFSPath {
	bfsPath := BFSPath{
		g:      graph,
		marked: map[Vertex]struct{}{},
		edgeTo: map[Vertex]Vertex{},
		distTo: map[Vertex]int{},
	}
	bfsPath.bfs(start, sccSet)
	return &bfsPath
}

func toSet(sccSet []Vertex) map[Vertex]struct{} {
	set := map[Vertex]struct{}{}
	for _, v := range sccSet {
		set[v] = struct{}{}
	}
	return set
}

func (path *BFSPath) bfs(start Vertex, set map[Vertex]struct{}) {
	bfsQueue := make([]Vertex, 0)
	bfsQueue = append(bfsQueue, start)
	path.marked[start] = struct{}{}
	path.distTo[start] = 0

	for len(bfsQueue) != 0 {
		currentV := bfsQueue[0]
		bfsQueue = bfsQueue[1:]
		for _, v := range path.g.In(currentV) {
			if _, inset := set[v]; !inset {
				continue
			}
			_, marked := path.marked[v]
			if !marked {
				path.edgeTo[v] = currentV
				path.distTo[v] = path.distTo[currentV] + 1
				path.marked[v] = struct{}{}
				bfsQueue = append(bfsQueue, v)
			}
		}
	}
}

// HasPathFrom ...
func (path *BFSPath) HasPathFrom(vertex Vertex) bool {
	_, hasPath := path.marked[vertex]
	return hasPath
}

// DistFrom ...
func (path *BFSPath) DistFrom(vertex Vertex) int {
	length, hasPath := path.distTo[vertex]
	if !hasPath {
		return -1
	}
	return length
}

// CycleTrace records a cycle path
type CycleTrace struct {
	from Vertex
	Rels []Rel
}

// PathFrom ...
func (path *BFSPath) PathFrom(vertex Vertex) []CycleTrace {
	if !path.HasPathFrom(vertex) {
		return nil
	}
	var paths []CycleTrace
	var current Vertex

	for current = vertex; path.distTo[current] != 0; current = path.edgeTo[current] {
		next := path.edgeTo[current]

		paths = append(paths, CycleTrace{
			from: current,
			Rels: getRelsFromEdges(path.g.Edges(current, next)),
		})
	}
	return append(paths, CycleTrace{
		from: current,
		Rels: getRelsFromEdges(path.g.Edges(current, vertex)),
	})
}

func getRelsFromEdges(es []Edge) (rs []Rel) {
	for _, e := range es {
		rs = append(rs, e.Value)
	}
	return rs
}
func getVerticesFromTracePath(ts []CycleTrace) (vs []Vertex) {
	for _, t := range ts {
		vs = append(vs, t.from)
	}
	return vs
}
