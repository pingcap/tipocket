package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// (deftest process-graph-test
//   (let [o1 {:index 0 :process 1 :type :ok}
//         o2 {:index 1 :process 2 :type :ok}
//         o3 {:index 2 :process 2 :type :ok}
//         o4 {:index 3 :process 1 :type :ok}
//         history [o1 o2 o3 o4]]
//     (is (= {o1 #{o4}, o2 #{o3}, o3 #{}, o4 #{}}
//            (g/->clj (:graph (process-graph history)))))))
func TestProcessGraph(t *testing.T) {
	history, err := ParseHistory(`{:index 0 :process 1 :type :ok}
{:index 1 :process 2 :type :ok}
{:index 2 :process 2 :type :ok}
{:index 3 :process 1 :type :ok}`)

	var (
		v0 = Vertex{Value: history[0]}
		v1 = Vertex{Value: history[1]}
		v2 = Vertex{Value: history[2]}
		v3 = Vertex{Value: history[3]}
	)
	processGraph := DirectedGraph{
		Outs: map[Vertex]map[Vertex][]Rel{
			v0: {
				v3: {Process},
			},
			v1: {
				v2: {Process},
			},
		},
	}

	g, _ := ProcessGraph(history)
	assert.Equal(t, err, nil, "test process graph, parse history")
	assert.Equal(t, g, processGraph)
}
