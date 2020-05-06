package list_append

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

func TestWWGraph(t *testing.T) {
	t1, _ := core.ParseOp(`{:type :ok, :value [[:append x 1]]}`)
	t2, _ := core.ParseOp(`{:type :ok, :value [[:append x 2]]}`)
	t3, _ := core.ParseOp(`{:type :ok, :value [[:r x [1 2]]]}`)

	_, g, _ := wwGraph([]core.Op{t1, t2, t3})

	expect := core.NewDirectedGraph()
	expect.Link(core.Vertex{t1}, core.Vertex{t2}, core.WW)

	require.Equal(t, expect, g)
}

func TestWRGraph(t *testing.T) {
	t1, _ := core.ParseOp(`{:type :ok, :value [[:append x 1]]}`)
	t2, _ := core.ParseOp(`{:type :ok, :value [[:r x [1]]]}`)
	t3, _ := core.ParseOp(`{:type :ok, :value [[:append x 2]]}`)
	t4, _ := core.ParseOp(`{:type :ok, :value [[:r x [1 2]]]}`)

	_, g, _ := wrGraph([]core.Op{t1, t2, t3, t4})

	expect := core.NewDirectedGraph()
	expect.Link(core.Vertex{t1}, core.Vertex{t2}, core.WR)
	expect.Link(core.Vertex{t3}, core.Vertex{t4}, core.WR)

	require.Equal(t, expect, g)
}

func TestGraph(t *testing.T) {
	ax1, _ := core.ParseOp(`{:type :ok, :value [[:append x 1]]}`)
	ax2, _ := core.ParseOp(`{:type :ok, :value [[:append x 2]]}`)
	//ax1ay1, _ := core.ParseOp(`{:type :ok, :value [[:append :x 1] [:append :y 1]]}`)
	//ax1ry1, _ := core.ParseOp(`{:type :ok, :value [[:append :x 1] [:r :y [1]]]}`)
	//ax2ay1, _ := core.ParseOp(`{:type :ok, :value [[:append :x 2] [:append :y 1]]}`)
	//ax2ay2, _ := core.ParseOp(`{:type :ok, :value [[:append :x 2] [:append :y 2]]}`)
	az1ax1ay1, _ := core.ParseOp(`{:type :ok, :value [[:append z 1] [:append x 1] [:append y 1]]}`)
	rxay1, _ := core.ParseOp(`{:type :ok, :value [[:r x nil] [:append y 1]]}`)
	ryax1, _ := core.ParseOp(`{:type :ok, :value [[:r y nil] [:append x 1]]}`)
	//rx121, _ := core.ParseOp(`{:type :ok, :value [[:r :x [1 2 1]]]}`)
	rx1ry1, _ := core.ParseOp(`{:type :ok, :value [[:r x [1]] [:r y [1]]]}`)
	rx1ay2, _ := core.ParseOp(`{:type :ok, :value [[:r x [1]] [:append y 2]]}`)
	ry12az3, _ := core.ParseOp(`{:type :ok, :value [[:r y [1 2]] [:append z 3]]}`)
	rz13, _ := core.ParseOp(`{:type :ok, :value [[:r z [1 3]]]}`)
	rx, _ := core.ParseOp(`{:type :ok, :value [[:r x nil]]}`)
	rx1, _ := core.ParseOp(`{:type :ok, :value [[:r x [1]]]}`)
	rx12, _ := core.ParseOp(`{:type :ok, :value [[:r x [1 2]]]}`)
	//rx12ry1, _ := core.ParseOp(`{:type :ok, :value [[:r :x [1 2]] [:r :y [1]]]}`)
	//rx12ry21, _ := core.ParseOp(`{:type :ok, :value [[:r :x [1 2]] [:r :y [2 1]]]}`)

	if true {
		_, g, _ := graph([]core.Op{ax1, rx1})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{ax1}, core.Vertex{rx1}, core.WR)
		require.Equal(t, expect.Outs, g.Outs)
	}

	if true {
		_, g, _ := graph([]core.Op{rx, ax1, rx1})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{rx}, core.Vertex{ax1}, core.RW)
		expect.Link(core.Vertex{ax1}, core.Vertex{rx1}, core.WR)
		require.Equal(t, expect.Outs, g.Outs)
	}

	if true {
		_, g, _ := graph([]core.Op{ax2, ax1, rx12})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{ax1}, core.Vertex{ax2}, core.WW)
		expect.Link(core.Vertex{ax2}, core.Vertex{rx12}, core.WR)
		require.Equal(t, expect.Outs, g.Outs)
	}

	if true {
		_, g, _ := graph([]core.Op{az1ax1ay1, rx1ay2, ry12az3, rz13})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{az1ax1ay1}, core.Vertex{rx1ay2}, core.WW)
		expect.Link(core.Vertex{az1ax1ay1}, core.Vertex{ry12az3}, core.WW)

		expect.Link(core.Vertex{az1ax1ay1}, core.Vertex{rx1ay2}, core.WR)
		expect.Link(core.Vertex{rx1ay2}, core.Vertex{ry12az3}, core.WR)

		expect.Link(core.Vertex{ry12az3}, core.Vertex{rz13}, core.WR)
		require.Equal(t, expect.Outs, g.Outs)
	}

	if true {
		t1, _ := core.ParseOp(`{:type :ok, :value [[:append x 1] [:append y 1]]}`)
		t2, _ := core.ParseOp(`{:type :ok, :value [[:append x 2] [:append y 2]]}`)
		t3, _ := core.ParseOp(`{:type :ok, :value [[:r x [1 2]] [:r y [2 1]]]}`)

		_, g, _ := graph([]core.Op{t1, t2, t3})
		expect := core.NewDirectedGraph()

		expect.Link(core.Vertex{t1}, core.Vertex{t2}, core.WW)
		expect.Link(core.Vertex{t1}, core.Vertex{t3}, core.WR)

		expect.Link(core.Vertex{t2}, core.Vertex{t1}, core.WW)
		expect.Link(core.Vertex{t2}, core.Vertex{t3}, core.WR)

		require.Equal(t, expect.Outs, g.Outs)
	}
	if true {
		checkResult := core.Check(graph, []core.Op{rxay1, ryax1, rx1ry1})
		require.Equal(t, 1, len(checkResult.Sccs))
		if !reflect.DeepEqual([]string{`Let:
  T1 = {:type ok :value [[:r y nil] [:append x 1]]
  T2 = {:type ok :value [[:r x nil] [:append y 1]]

Then:
  - T1 < T2, because T1 observed the initial (nil) state of y, which T2 created by appending 1.
  - However, T2 < T1, because T2 observed the initial (nil) state of x, which T1 created by appending 1: a contradiction!`}, checkResult.Cycles) {
			require.Equal(t, `Let:
  T1 = {:type ok :value [[:r x nil] [:append y 1]]
  T2 = {:type ok :value [[:r y nil] [:append x 1]]

Then:
  - T1 < T2, because T1 observed the initial (nil) state of x, which T2 created by appending 1.
  - However, T2 < T1, because T2 observed the initial (nil) state of y, which T1 created by appending 1: a contradiction!`, checkResult.Cycles)
		}
	}

	if true {
		t1, _ := core.ParseOp(`{:type :info, :value [[:append x 2] [:r y nil]]}`)
		t2, _ := core.ParseOp(`{:type :ok, :value [[:append x 1] [:append y 1]]}`)
		t3, _ := core.ParseOp(`{:type :ok, :value [[:r x [1 2]] [:r y [1]]]}`)

		_, g, _ := graph([]core.Op{t1, t2, t3})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{t1}, core.Vertex{t3}, core.WR)
		expect.Link(core.Vertex{t2}, core.Vertex{t1}, core.WW)
		expect.Link(core.Vertex{t2}, core.Vertex{t3}, core.WR)

		require.Equal(t, expect.Outs, g.Outs)
	}

	if true {
		_, g, _ := graph([]core.Op{rx, ax1})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{rx}, core.Vertex{ax1}, core.RW)
		require.Equal(t, expect.Outs, g.Outs)

		_, g, _ = graph([]core.Op{rx, ax1, ax2})
		require.Equal(t, core.NewDirectedGraph(), g)
	}

	if true {
		defer func() {
			expect := "duplicate appends, op {:type invoke :value [[:append y 2] [:append x 1]] :time 1, key: x, value: 1"
			if r := recover(); r == nil || r.(string) != expect {
				t.Fatalf("expect got panic %s", expect)
			}
		}()
		t := func() {
			ax1ry, _ := core.ParseOp(`{:index 0, :type :invoke, :value [[:append x 1] [:r y nil]]}`)
			ay2ax1, _ := core.ParseOp(`{:index 1, :type :invoke, :value [[:append y 2] [:append x 1]]}`)
			graph([]core.Op{ax1ry, ay2ax1})
		}
		t()
	}
}

func TestG1aCases(t *testing.T) {
	t1, _ := core.ParseOp(`{:type :fail, :value [[:append x 1]]}`)
	t2, _ := core.ParseOp(`{:type :ok, :value [[:r x [1]] [:append x 2]]}`)
	t3, _ := core.ParseOp(`{:type :ok, :value [[:r x [1 2]] [:r y [3]]]}`)

	got := g1aCases([]core.Op{t2, t3, t1})
	expect := GCaseTp{G1Conflict{
		Op: t2,
		Mop: core.Read{
			Key:   "x",
			Value: []int{1},
		},
		Writer:  t1,
		Element: 1,
	}, G1Conflict{
		Op: t3,
		Mop: core.Read{
			Key:   "x",
			Value: []int{1, 2},
		},
		Writer:  t1,
		Element: 1,
	}}

	require.Equal(t, expect, got)
}

func TestCheck(t *testing.T) {
	var history = core.History{
		core.Op{Type: core.OpTypeOk,
			Value: &[]core.Mop{
				core.Append{
					Key:   "x",
					Value: 1,
				},
				core.Read{
					Key:   "y",
					Value: []int{1},
				},
			}},
		core.Op{Type: core.OpTypeOk,
			Value: &[]core.Mop{
				core.Append{
					Key:   "x",
					Value: 2,
				},
				core.Append{
					Key:   "y",
					Value: 1,
				}}},
		core.Op{Type: core.OpTypeOk,
			Value: &[]core.Mop{
				core.Read{
					Key:   "x",
					Value: []int{1, 2},
				}}},
	}

	result := Check(txn.Opts{
		ConsistencyModels: []core.ConsistencyModelName{"serializable"},
	}, history)

	fmt.Printf("%#v", result)
}

func TestHugeScc(t *testing.T) {
	content, err := ioutil.ReadFile("../histories/huge-scc.edn")
	if err != nil {
		t.Fail()
	}
	history, err := core.ParseHistory(string(content))
	if err != nil {
		t.Fail()
	}
	result := Check(txn.Opts{}, history)
	fmt.Printf("%#v", result)
}
