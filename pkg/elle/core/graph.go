package core

type DirectedGraph struct{}
type SCC struct{}
type Transition struct{}
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
