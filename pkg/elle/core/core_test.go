package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcessGraph(t *testing.T) {
	history, err := ParseHistory(`{:index 0 :process 1 :type :ok}
{:index 1 :process 2 :type :ok}
{:index 2 :process 2 :type :ok}
{:index 3 :process 1 :type :ok}`)

	assert.Equal(t, err, nil, "test process graph, parse history")

	var (
		v0 = Vertex{Value: history[0]}
		v1 = Vertex{Value: history[1]}
		v2 = Vertex{Value: history[2]}
		v3 = Vertex{Value: history[3]}
	)
	processGraph := NewDirectedGraph()
	processGraph.Outs = map[Vertex]map[Vertex][]Rel{
		v0: {
			v3: {Process},
		},
		v1: {
			v2: {Process},
		},
	}

	_, g, _ := ProcessGraph(history)

	var graphOuts map[Vertex]map[Vertex][]Rel = map[Vertex]map[Vertex][]Rel{}
	for k, v := range g.Outs {
		if len(v) != 0 {
			v, e := processGraph.Outs[k]
			if !e {
				continue
			}
			graphOuts[k] = v
		}
	}

	assert.Equal(t, graphOuts, processGraph.Outs)
}

//func toJson(v interface{}) string {
//	s, err := json.MarshalIndent(v, "", "\t")
//	if err != nil {
//		panic(err)
//	}
//	return string(s)
//}
