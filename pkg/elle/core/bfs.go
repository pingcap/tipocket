package core

type BFSPath struct {
	marked map[Vertex]struct{}
	edgeTo map[Vertex]Vertex
	distTo map[Vertex]int
}

func NewBFSPath(graph *DirectedGraph, start Vertex, sccSet map[Vertex]struct{}) *BFSPath {
	bfsPath := BFSPath{
		marked: map[Vertex]struct{}{},
		edgeTo: map[Vertex]Vertex{},
		distTo: map[Vertex]int{},
	}
	bfsPath.bfs(graph, start, sccSet)
	return &bfsPath
}

func toSet(sccSet []Vertex) map[Vertex]struct{} {
	set := map[Vertex]struct{}{}
	for _, v := range sccSet {
		set[v] = struct{}{}
	}
	return set
}

func (path *BFSPath) bfs(graph *DirectedGraph, start Vertex, set map[Vertex]struct{}) {
	bfsQueue := make([]Vertex, 0)
	bfsQueue = append(bfsQueue, start)
	path.marked[start] = struct{}{}
	path.distTo[start] = 0

	for len(bfsQueue) != 0 {
		currentV := bfsQueue[0]
		bfsQueue = bfsQueue[1:]
		for _, v := range graph.In(currentV) {
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

func (path *BFSPath) HasPathTo(vertex Vertex) bool {
	_, hasPath := path.marked[vertex]
	return hasPath
}

func (path *BFSPath) DistTo(vertex Vertex) int {
	length, hasPath := path.distTo[vertex]
	if !hasPath {
		return -1
	}
	return length
}

func (path *BFSPath) PathTo(vertex Vertex) []Vertex {
	if !path.HasPathTo(vertex) {
		return nil
	}
	var paths []Vertex
	var current Vertex
	for current = vertex; path.distTo[current] != 0; current = path.edgeTo[current] {
		paths = append(paths, current)
	}
	return append(paths, current)
}
