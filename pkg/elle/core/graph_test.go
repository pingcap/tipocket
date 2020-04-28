package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindCycle1(t *testing.T) {
	var g DirectedGraph
	g.Ins = make(map[Vertex][]Vertex)
	g.Outs = make(map[Vertex]map[Vertex][]Rel)

	g.Link(Vertex{1}, Vertex{2}, "")
	g.Link(Vertex{2}, Vertex{3}, "")
	g.Link(Vertex{3}, Vertex{4}, "")
	g.Link(Vertex{4}, Vertex{5}, "")
	g.Link(Vertex{5}, Vertex{6}, "")
	g.Link(Vertex{6}, Vertex{4}, "")
	g.Link(Vertex{6}, Vertex{1}, "")

	cycle := FindCycle(g, SCC{
		Vertices: []Vertex{Vertex{4}, Vertex{5}, Vertex{6}},
	})

	assert.Equal(t, cycle, []Vertex{Vertex{4}, Vertex{5}, Vertex{6}, Vertex{4}}, "cycle")
}

func TestFindCycle2(t *testing.T) {
	var g DirectedGraph
	g.Ins = make(map[Vertex][]Vertex)
	g.Outs = make(map[Vertex]map[Vertex][]Rel)

	g.Link(Vertex{1}, Vertex{2}, "")
	g.Link(Vertex{2}, Vertex{3}, "")
	g.Link(Vertex{3}, Vertex{4}, "")
	g.Link(Vertex{4}, Vertex{5}, "")
	g.Link(Vertex{5}, Vertex{6}, "")
	g.Link(Vertex{6}, Vertex{4}, "")
	g.Link(Vertex{6}, Vertex{1}, "")

	cycle := FindCycle(g, SCC{
		Vertices: []Vertex{Vertex{1}, Vertex{2}, Vertex{3}, Vertex{4}, Vertex{5}, Vertex{6}},
	})

	assert.Equal(t, cycle, []Vertex{Vertex{4}, Vertex{5}, Vertex{6}, Vertex{4}}, "cycle")
}

func TestFindCycle3(t *testing.T) {
	var g DirectedGraph
	g.Ins = make(map[Vertex][]Vertex)
	g.Outs = make(map[Vertex]map[Vertex][]Rel)

	g.Link(Vertex{1}, Vertex{2}, "")
	g.Link(Vertex{2}, Vertex{1}, "")

	cycle := FindCycle(g, SCC{
		Vertices: []Vertex{Vertex{1}, Vertex{2}},
	})

	assert.Equal(t, cycle, []Vertex{Vertex{1}, Vertex{2}, Vertex{1}}, "cycle")
}

func TestFindCycleStartingWith1(t *testing.T) {
	var g DirectedGraph
	g.Ins = make(map[Vertex][]Vertex)
	g.Outs = make(map[Vertex]map[Vertex][]Rel)

	g.Link(Vertex{1}, Vertex{2}, "start1")
	g.Link(Vertex{2}, Vertex{3}, "")
	g.Link(Vertex{3}, Vertex{4}, "")
	g.Link(Vertex{4}, Vertex{5}, "start4")
	g.Link(Vertex{5}, Vertex{6}, "")
	g.Link(Vertex{6}, Vertex{4}, "")
	g.Link(Vertex{6}, Vertex{1}, "")

	cycle := FindCycleStartingWith(g, "start4", SCC{
		Vertices: []Vertex{Vertex{1}, Vertex{2}, Vertex{3}, Vertex{4}, Vertex{5}, Vertex{6}},
	})
	assert.Equal(t, cycle, []Vertex{Vertex{4}, Vertex{5}, Vertex{6}, Vertex{4}}, "cycle")
}

func TestFindCycleStartingWith2(t *testing.T) {
	var g DirectedGraph
	g.Ins = make(map[Vertex][]Vertex)
	g.Outs = make(map[Vertex]map[Vertex][]Rel)

	g.Link(Vertex{1}, Vertex{2}, "start1")
	g.Link(Vertex{2}, Vertex{3}, "")
	g.Link(Vertex{3}, Vertex{4}, "")
	g.Link(Vertex{4}, Vertex{5}, "start4")
	g.Link(Vertex{5}, Vertex{6}, "")
	g.Link(Vertex{6}, Vertex{4}, "")
	g.Link(Vertex{6}, Vertex{1}, "")

	cycle := FindCycleStartingWith(g, "start1", SCC{
		Vertices: []Vertex{Vertex{1}, Vertex{2}, Vertex{3}, Vertex{4}, Vertex{5}, Vertex{6}},
	})
	assert.Equal(t, cycle, []Vertex{Vertex{1}, Vertex{2}, Vertex{3}, Vertex{4}, Vertex{5}, Vertex{6}, Vertex{1}}, "cycle")
}

func TestFindCycleStartingWith3(t *testing.T) {
	var g DirectedGraph
	g.Ins = make(map[Vertex][]Vertex)
	g.Outs = make(map[Vertex]map[Vertex][]Rel)

	g.Link(Vertex{1}, Vertex{2}, "start1")
	g.Link(Vertex{2}, Vertex{3}, "")
	g.Link(Vertex{3}, Vertex{4}, "")
	g.Link(Vertex{4}, Vertex{5}, "start4")
	g.Link(Vertex{5}, Vertex{6}, "")
	g.Link(Vertex{6}, Vertex{4}, "")
	g.Link(Vertex{6}, Vertex{1}, "")

	cycle := FindCycleStartingWith(g, "", SCC{
		Vertices: []Vertex{Vertex{1}, Vertex{2}, Vertex{3}, Vertex{4}, Vertex{5}, Vertex{6}},
	})
	assert.Equal(t, cycle, []Vertex{Vertex{2}, Vertex{3}, Vertex{4}, Vertex{5}, Vertex{6}, Vertex{1}, Vertex{2}}, "cycle")
}
