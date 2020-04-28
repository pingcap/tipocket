package core

import (
	"encoding/json"
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

func TestRealtimeGraph(t *testing.T) {
	history, err := ParseHistory(`{:type :invoke :process 1 :f :read :value nil}
{:type :ok      :process 1 :f :read :value 1}
{:type :invoke :process 2 :f :read :value nil}
{:type :ok      :process 2 :f :read :value 2}
{:type :invoke :process 3 :f :read :value nil}
{:type :ok      :process 3 :f :read :value 3}
{:type :invoke :process 4 :f :read :value nil}
{:type :ok      :process 4 :f :read :value 4}
{:type :invoke :process 5 :f :read :value nil}
{:type :ok      :process 5 :f :read :value 5}`)

	assert.Equal(t, err, nil, "test process graph, parse history")
	assert.Equal(t, len(history), 10, "length of history error")

	call1, resp1 := history[0], history[1]
	call2, resp2 := history[2], history[3]
	call3, resp3 := history[4], history[5]
	call4, resp4 := history[6], history[7]
	call5, resp5 := history[8], history[9]
	// Test	empty
	_, g, _ := RealtimeGraph(History{})
	if len(g.Ins) != 0 || len(g.Outs) != 0 {
		assert.Fail(t, "empty history should return empty graph")
	}

	_, g, _ = RealtimeGraph(History{call1, resp1})

	assert.Equal(t, g.Outs, map[Vertex]map[Vertex][]Rel{})

	_, g, _ = RealtimeGraph(History{call1, resp1, call2, resp2})
	dest := map[Vertex]map[Vertex][]Rel{
		Vertex{Value: resp1}: {
			Vertex{Value: resp2}: []Rel{Realtime},
		},
		Vertex{Value: resp2}: {},
	}

	assert.Equal(t, g.Outs, dest)

	_, g, _ = RealtimeGraph(History{call1, resp1, call2, resp2, call3, resp3})

	dest = map[Vertex]map[Vertex][]Rel{
		Vertex{Value: resp1}: {
			Vertex{Value: resp2}: []Rel{Realtime},
		},
		Vertex{Value: resp2}: {
			Vertex{Value: resp3}: []Rel{Realtime},
		},
		Vertex{Value: resp3}: {},
	}

	assert.Equal(t, g.Outs, dest)

	_, g, _ = RealtimeGraph(History{call1, resp1, call2, call3, resp3, resp2})

	dest = map[Vertex]map[Vertex][]Rel{
		Vertex{Value: resp1}: {
			Vertex{Value: resp2}: []Rel{Realtime},
			Vertex{Value: resp3}: []Rel{Realtime},
		},
		Vertex{Value: resp2}: {},
		Vertex{Value: resp3}: {},
	}

	assert.Equal(t, g.Outs, dest)

	_, g, _ = RealtimeGraph(History{call1, call2, resp2, resp1, call3, call4, resp3, resp4})

	dest = map[Vertex]map[Vertex][]Rel{
		Vertex{Value: resp1}: {
			Vertex{Value: resp4}: []Rel{Realtime},
			Vertex{Value: resp3}: []Rel{Realtime},
		},
		Vertex{Value: resp2}: {
			Vertex{Value: resp3}: []Rel{Realtime},
			Vertex{Value: resp4}: []Rel{Realtime},
		},
		Vertex{Value: resp3}: {},
		Vertex{Value: resp4}: {},
	}

	assert.Equal(t, g.Outs, dest)

	_, g, _ = RealtimeGraph(History{call1, resp1, call2, call4, resp2, call3, resp4, resp3, call5, resp5})

	dest = map[Vertex]map[Vertex][]Rel{
		Vertex{Value: resp1}: {
			Vertex{Value: resp4}: []Rel{Realtime},
			Vertex{Value: resp2}: []Rel{Realtime},
		},
		Vertex{Value: resp2}: {
			Vertex{Value: resp3}: []Rel{Realtime},
		},
		Vertex{Value: resp3}: {
			Vertex{Value: resp5}: []Rel{Realtime},
		},
		Vertex{Value: resp4}: {
			Vertex{Value: resp5}: []Rel{Realtime},
		},
		Vertex{Value: resp5}: {},
	}

	assert.Equal(t, g.Outs, dest)

}

func TestMopValueType(t *testing.T) {
	//	history, err := ParseHistory(`{:type :invoke, :f :txn, :value [[:append 11 1] [:r 11 nil] [:r 11 nil]], :process 10, :time 14532933, :index 0}
	//{:type :invoke, :f :txn, :value [[:append 9 1]], :process 18, :time 16357277, :index 1}
	//{:type :ok, :f :txn, :value [[:append 11 1] [:r 11 [1]] [:r 11 [1]]], :process 10, :time 31076248, :index 2}
	//{:type :ok, :f :txn, :value [[:append 9 1]], :process 18, :time 40237832, :index 3}
	//{:type :invoke, :f :txn, :value [[:r 11 nil] [:r 9 nil]], :process 13, :time 40623375, :index 4}
	//{:type :invoke, :f :txn, :value [[:append 10 1] [:r 8 nil] [:append 10 2]], :process 20, :time 40896524, :index 5}
	//{:type :invoke, :f :txn, :value [[:r 11 nil] [:append 7 1] [:r 10 nil] [:r 11 nil]], :process 10, :time 53339097, :index 6}
	//{:type :invoke, :f :txn, :value [[:r 8 nil]], :process 18, :time 56451006, :index 7}
	//{:type :invoke, :f :txn, :value [[:r 11 nil] [:append 11 2] [:r 10 nil] [:r 11 nil]], :process 24, :time 57053179, :index 8}
	//{:type :ok, :f :txn, :value [[:r 11 [1]] [:r 9 [1]]], :process 13, :time 58266815, :index 9}
	//{:type :ok, :f :txn, :value [[:append 10 1] [:r 8 []] [:append 10 2]], :process 20, :time 66867075, :index 10}
	//{:type :invoke, :f :txn, :value [[:append 11 3] [:r 11 nil] [:r 9 nil] [:r 9 nil]], :process 4, :time 66786847, :index 11}
	//{:type :ok, :f :txn, :value [[:r 8 []]], :process 18, :time 67064893, :index 12}
	//{:type :ok, :f :txn, :value [[:r 11 [1]] [:append 11 2] [:r 10 [1 2]] [:r 11 [1 2]]], :process 24, :time 75478039, :index 13}
	//{:type :ok, :f :txn, :value [[:r 11 [1 2]] [:append 7 1] [:r 10 [1 2]] [:r 11 [1 2]]], :process 10, :time 79883607, :index 14}
	//{:type :ok, :f :txn, :value [[:append 11 3] [:r 11 [1 2 3]] [:r 9 [1]] [:r 9 [1]]], :process 4, :time 88446473, :index 15}
	//{:type :invoke, :f :txn, :value [[:r 11 nil]], :process 8, :time 88424151, :index 16}`)

	history, err := ParseHistory(`{:type :invoke, :f :txn, :value [[:append 11 1] [:r 11 nil] [:r 11 nil]], :process 10 :time 14532933, :index 0}
{:type :invoke, :f :txn, :value [[:append 9 1]], :process 18 :time 16357277, :index 1}
{:type :ok, :f :txn, :value [[:append 11 1] [:r 11 [1]] [:r 11 [1]]], :process 10 :time 31076248, :index 2}
{:type :invoke, :f :txn, :value [[:r 11 nil]], :process 8 :time 88424151, :index 16}`)

	if err != nil {
		assert.Fail(t, "Parse history failed", err)
	}

	for _, op := range history {
		if op.Value == nil {
			continue
		}
		for _, mop := range *op.Value {
			if mop.IsAppend() {
				arg := mop.GetValue().(MopValueType)
				_ = arg.(int)
			} else {
				if mop.GetValue() == nil {
					continue
				}
				args := mop.GetValue().([]int)
				for _ = range args {
					//_ = arg.(int)
				}
			}
		}
	}
}

// toJson is a debugging function, which can be used like:
// ```
//for k, v := range g.Outs {
//	fmt.Println(toJson(k), len(v))
//}
//
//fmt.Println()
//fmt.Println()
//
//for k, v := range dest {
//	fmt.Println(toJson(k), len(v))
//}
// ```
func toJson(v interface{}) string {
	s, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		panic(err)
	}
	return string(s)
}

// Note: MonotonicKeyGraph requires rw_register, which is not supported now.
//func TestCheck(t *testing.T) {
//	// testing valid
//	history, err := ParseHistory(`{:index 0 :type :invoke :process 0 :f :read :value nil}
//                   {:index 1 :type :ok     :process 0 :f :read :value {:x 0 :y 0}}
//                   {:index 2 :type :invoke :process 0 :f :inc :value [:x]}
//                   {:index 3 :type :ok     :process 0 :f :inc :value {:x 1}}
//                   {:index 4 :type :invoke :process 0 :f :read :value nil}
//                   {:index 5 :type :ok     :process 0 :f :read :value {:x 1 :y 1}}`)
//	if err != nil {
//		assert.Equal(t, err, nil, "test process graph, parse history")
//	}
//	assert.Equal(t, len(history), 6, "length of history should be 6")
//
//	res := Check(MonotonicKeyGraph, history)
//	assert.Equal(t, 0, len(res.Sccs), "length of sccs should be zero")
//}
