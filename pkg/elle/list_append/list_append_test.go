package listappend

import (
	"io/ioutil"
	"log"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

func TestWWGraph(t *testing.T) {
	t1 := mustParseOp(`{:type :ok, :value [[:append x 1]]}`)
	t2 := mustParseOp(`{:type :ok, :value [[:append x 2]]}`)
	t3 := mustParseOp(`{:type :ok, :value [[:append x 3]]}`)
	t4 := mustParseOp(`{:type :ok, :value [[:r x [1 2 3]]]}`)

	_, g, _ := wwGraph([]core.Op{t1, t2, t3, t4})

	expect := core.NewDirectedGraph()
	expect.Link(core.Vertex{Value: t1}, core.Vertex{Value: t2}, core.WW)
	expect.Link(core.Vertex{Value: t2}, core.Vertex{Value: t3}, core.WW)

	require.Equal(t, expect, g)
}

func TestWRGraph(t *testing.T) {
	t1 := mustParseOp(`{:type :ok, :value [[:append x 1]]}`)
	t2 := mustParseOp(`{:type :ok, :value [[:r x [1]]]}`)
	t3 := mustParseOp(`{:type :ok, :value [[:append x 2]]}`)
	t4 := mustParseOp(`{:type :ok, :value [[:r x [1 2]]]}`)

	_, g, _ := wrGraph([]core.Op{t1, t2, t3, t4})

	expect := core.NewDirectedGraph()
	expect.Link(core.Vertex{Value: t1}, core.Vertex{Value: t2}, core.WR)
	expect.Link(core.Vertex{Value: t3}, core.Vertex{Value: t4}, core.WR)

	require.Equal(t, expect, g)
}

func TestGraph(t *testing.T) {
	ax1 := mustParseOp(`{:type :ok, :value [[:append x 1]]}`)
	ax2 := mustParseOp(`{:type :ok, :value [[:append x 2]]}`)
	//ax1ay1:= mustParseOp(`{:type :ok, :value [[:append :x 1] [:append :y 1]]}`)
	//ax1ry1:= mustParseOp(`{:type :ok, :value [[:append :x 1] [:r :y [1]]]}`)
	//ax2ay1:= mustParseOp(`{:type :ok, :value [[:append :x 2] [:append :y 1]]}`)
	//ax2ay2:= mustParseOp(`{:type :ok, :value [[:append :x 2] [:append :y 2]]}`)
	az1ax1ay1 := mustParseOp(`{:type :ok, :value [[:append z 1] [:append x 1] [:append y 1]]}`)
	rxay1 := mustParseOp(`{:type :ok, :value [[:r x nil] [:append y 1]]}`)
	ryax1 := mustParseOp(`{:type :ok, :value [[:r y nil] [:append x 1]]}`)
	//rx121:= mustParseOp(`{:type :ok, :value [[:r :x [1 2 1]]]}`)
	rx1ry1 := mustParseOp(`{:type :ok, :value [[:r x [1]] [:r y [1]]]}`)
	rx1ay2 := mustParseOp(`{:type :ok, :value [[:r x [1]] [:append y 2]]}`)
	ry12az3 := mustParseOp(`{:type :ok, :value [[:r y [1 2]] [:append z 3]]}`)
	rz13 := mustParseOp(`{:type :ok, :value [[:r z [1 3]]]}`)
	rx := mustParseOp(`{:type :ok, :value [[:r x nil]]}`)
	rx1 := mustParseOp(`{:type :ok, :value [[:r x [1]]]}`)
	rx12 := mustParseOp(`{:type :ok, :value [[:r x [1 2]]]}`)
	//rx12ry1:= mustParseOp(`{:type :ok, :value [[:r :x [1 2]] [:r :y [1]]]}`)
	//rx12ry21:= mustParseOp(`{:type :ok, :value [[:r :x [1 2]] [:r :y [2 1]]]}`)

	switches := true

	if switches {
		_, g, _ := graph([]core.Op{ax1, rx1})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{Value: ax1}, core.Vertex{Value: rx1}, core.WR)
		require.Equal(t, expect.Outs, g.Outs)
	}

	if switches {
		_, g, _ := graph([]core.Op{rx, ax1, rx1})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{Value: rx}, core.Vertex{Value: ax1}, core.RW)
		expect.Link(core.Vertex{Value: ax1}, core.Vertex{Value: rx1}, core.WR)
		require.Equal(t, expect.Outs, g.Outs)
	}

	if switches {
		_, g, _ := graph([]core.Op{ax2, ax1, rx12})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{Value: ax1}, core.Vertex{Value: ax2}, core.WW)
		expect.Link(core.Vertex{Value: ax2}, core.Vertex{Value: rx12}, core.WR)
		require.Equal(t, expect.Outs, g.Outs)
	}

	if switches {
		_, g, _ := graph([]core.Op{az1ax1ay1, rx1ay2, ry12az3, rz13})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{Value: az1ax1ay1}, core.Vertex{Value: rx1ay2}, core.WW)
		expect.Link(core.Vertex{Value: az1ax1ay1}, core.Vertex{Value: ry12az3}, core.WW)

		expect.Link(core.Vertex{Value: az1ax1ay1}, core.Vertex{Value: rx1ay2}, core.WR)
		expect.Link(core.Vertex{Value: rx1ay2}, core.Vertex{Value: ry12az3}, core.WR)

		expect.Link(core.Vertex{Value: ry12az3}, core.Vertex{Value: rz13}, core.WR)
		require.Equal(t, expect.Outs, g.Outs)
	}

	if switches {
		t1 := mustParseOp(`{:type :ok, :value [[:append x 1] [:append y 1]]}`)
		t2 := mustParseOp(`{:type :ok, :value [[:append x 2] [:append y 2]]}`)
		t3 := mustParseOp(`{:type :ok, :value [[:r x [1 2]] [:r y [2 1]]]}`)

		_, g, _ := graph([]core.Op{t1, t2, t3})
		expect := core.NewDirectedGraph()

		expect.Link(core.Vertex{Value: t1}, core.Vertex{Value: t2}, core.WW)
		expect.Link(core.Vertex{Value: t1}, core.Vertex{Value: t3}, core.WR)

		expect.Link(core.Vertex{Value: t2}, core.Vertex{Value: t1}, core.WW)
		expect.Link(core.Vertex{Value: t2}, core.Vertex{Value: t3}, core.WR)

		require.Equal(t, expect.Outs, g.Outs)
	}

	if switches {
		checkResult := core.Check(graph, []core.Op{rxay1, ryax1, rx1ry1})
		require.Equal(t, 1, len(checkResult.Sccs))
		if !reflect.DeepEqual([]string{`Let:
  T1 = {:type :ok, :value [[:r y nil] [:append x 1]]}
  T2 = {:type :ok, :value [[:r x nil] [:append y 1]]}

Then:
  - T1 < T2, because T1 observed the initial (nil) state of y, which T2 created by appending 1.
  - However, T2 < T1, because T2 observed the initial (nil) state of x, which T1 created by appending 1: a contradiction!`}, checkResult.Cycles) {
			require.Equal(t, []string{`Let:
  T1 = {:type :ok, :value [[:r x nil] [:append y 1]]}
  T2 = {:type :ok, :value [[:r y nil] [:append x 1]]}

Then:
  - T1 < T2, because T1 observed the initial (nil) state of x, which T2 created by appending 1.
  - However, T2 < T1, because T2 observed the initial (nil) state of y, which T1 created by appending 1: a contradiction!`}, checkResult.Cycles)
		}
	}

	if switches {
		t1 := mustParseOp(`{:type :info, :value [[:append x 2] [:r y nil]]}`)
		t2 := mustParseOp(`{:type :ok, :value [[:append x 1] [:append y 1]]}`)
		t3 := mustParseOp(`{:type :ok, :value [[:r x [1 2]] [:r y [1]]]}`)

		_, g, _ := graph([]core.Op{t1, t2, t3})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{Value: t1}, core.Vertex{Value: t3}, core.WR)
		expect.Link(core.Vertex{Value: t2}, core.Vertex{Value: t1}, core.WW)
		expect.Link(core.Vertex{Value: t2}, core.Vertex{Value: t3}, core.WR)

		require.Equal(t, expect.Outs, g.Outs)
	}

	if switches {
		_, g, _ := graph([]core.Op{rx, ax1})
		expect := core.NewDirectedGraph()
		expect.Link(core.Vertex{Value: rx}, core.Vertex{Value: ax1}, core.RW)
		require.Equal(t, expect.Outs, g.Outs)

		_, g, _ = graph([]core.Op{rx, ax1, ax2})
		require.Equal(t, core.NewDirectedGraph(), g)
	}

	if switches {
		defer func() {
			expect := "duplicate appends, op {:type :invoke, :value [[:append y 2] [:append x 1]], :index 1}, key: x, value: 1"
			if r := recover(); r == nil || r.(string) != expect {
				t.Fatalf("expect got panic %s", expect)
			}
		}()
		t := func() {
			ax1ry := mustParseOp(`{:index 0, :type :invoke, :value [[:append x 1] [:r y nil]]}`)
			ay2ax1 := mustParseOp(`{:index 1, :type :invoke, :value [[:append y 2] [:append x 1]]}`)
			graph([]core.Op{ax1ry, ay2ax1})
		}
		t()
	}
}

func TestG1aCases(t *testing.T) {
	require.Equal(t, GCaseTp{}, g1aCases([]core.Op{}))

	t1 := mustParseOp(`{:type :fail, :value [[:append x 1]]}`)
	t2 := mustParseOp(`{:type :ok, :value [[:r x [1]] [:append x 2]]}`)
	t3 := mustParseOp(`{:type :ok, :value [[:r x [1 2]] [:r y [3]]]}`)

	got := g1aCases([]core.Op{t2, t3, t1})
	expect := GCaseTp{G1Conflict{
		Op:      t2,
		Mop:     core.Read("x", []int{1}),
		Writer:  t1,
		Element: 1,
	}, G1Conflict{
		Op:      t3,
		Mop:     core.Read("x", []int{1, 2}),
		Writer:  t1,
		Element: 1,
	}}

	require.Equal(t, expect, got)
}

func TestG1bCases(t *testing.T) {
	require.Equal(t, GCaseTp{}, g1bCases([]core.Op{}))

	t1 := mustParseOp(`{:type :ok, :value [[:append x 1] [:append x 2]]}`)
	t2 := mustParseOp(`{:type :ok, :value [[:r x [1]]]}`)
	t3 := mustParseOp(`{:type :ok, :value [[:r x [1 2]] [:r y [3]]]}`)
	t4 := mustParseOp(`{:type :ok, :value [[:r x [1 2 3]]]}`)

	got := g1bCases([]core.Op{t2, t3, t1, t4})

	expect := GCaseTp{G1Conflict{
		Op:      t2,
		Mop:     core.Read("x", []int{1}),
		Writer:  t1,
		Element: 1,
	}}

	require.Equal(t, expect, got)
}

func TestInternalCases(t *testing.T) {
	switches := true
	require.Equal(t, GCaseTp(nil), internalCases([]core.Op{}))
	if switches {
		t1 := mustParseOp(`{:type :ok, :value [[:r y [5 6]] [:append x 3] [:r x [1 2 3]] [:append x 4] [:r x [1 2 3 4]]]}]]}`)
		require.Equal(t, GCaseTp(nil), internalCases([]core.Op{t1}))
	}

	if switches {
		stale := mustParseOp(`{:type :ok, :value [[:r x [1 2]] [:append x 3] [:r x [1 2]]]}`)
		badPrefix := mustParseOp(`{:type :ok, :value [[:r x [1 2]] [:append x 3] [:r x [0 2 3]]]}`)
		extension := mustParseOp(`{:type :ok, :value [[:r x [1 2]] [:append x 3] [:r x [1 2 3 4]]]}`)
		shortRead := mustParseOp(`{:type :ok, :value [[:r x [1 2]] [:append x 3] [:r x [1]]]}`)

		got := internalCases([]core.Op{stale, badPrefix, extension, shortRead})
		expect := GCaseTp{
			InternalConflict{
				Op:       stale,
				Mop:      core.Read("x", []int{1, 2}),
				Expected: []int{1, 2, 3},
			},
			InternalConflict{
				Op:       badPrefix,
				Mop:      core.Read("x", []int{0, 2, 3}),
				Expected: []int{1, 2, 3},
			},
			InternalConflict{
				Op:       extension,
				Mop:      core.Read("x", []int{1, 2, 3, 4}),
				Expected: []int{1, 2, 3},
			},
			InternalConflict{
				Op:       shortRead,
				Mop:      core.Read("x", []int{1}),
				Expected: []int{1, 2, 3},
			},
		}
		require.Equal(t, expect, got)
	}
	if switches {
		disagreement := mustParseOp(`{:type :ok, :value [[:append x 3] [:r x [1 2 3 4]]]}`)
		shortRead := mustParseOp(`{:type :ok, :value [[:append x 3] [:r x []]]}`)

		got := internalCases([]core.Op{disagreement, shortRead})
		expect := GCaseTp{
			InternalConflict{
				Op:       disagreement,
				Mop:      core.Read("x", []int{1, 2, 3, 4}),
				Expected: []int{unknownPrefixMagicNumber, 3},
			},
			InternalConflict{
				Op:       shortRead,
				Mop:      core.Read("x", []int{}),
				Expected: []int{unknownPrefixMagicNumber, 3},
			},
		}
		require.Equal(t, expect, got)
	}
	if switches {
		t1 := mustParseOp(`{:type :invoke, :value [[:append 0 6] [:r 0 nil]] :process 1, :index 20}`)
		t2 := mustParseOp(`{:type :ok, :value [[:append 0 6] [:r 0 nil]] :process 1, :index 21}`)

		got := internalCases([]core.Op{t1, t2})

		expect := GCaseTp{
			InternalConflict{
				Op:       t2,
				Mop:      core.Read("0", nil),
				Expected: []int{unknownPrefixMagicNumber, 6},
			},
		}
		require.Equal(t, expect, got)
	}
}

func TestChecker(t *testing.T) {
	var switches = true
	// G0
	if switches {
		t1 := mustParseOp(`{:type :ok, :value [[:append x 1] [:append y 1]]}`)
		t2 := mustParseOp(`{:type :ok, :value [[:append x 2] [:append y 2]]}`)
		t3 := mustParseOp(`{:type :ok, :value [[:r x [1 2]] [:r y [2 1]]]}`)
		h := []core.Op{t1, t2, t3}

		expect := txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G0"},
			Anomalies: core.Anomalies{
				"G0": []core.Anomaly{
					core.CycleExplainerResult{
						Circle: core.Circle{
							Path: []core.Op{withIndex(t1, 0), withIndex(t2, 1), withIndex(t1, 0)},
						},
						Steps: []core.Step{{WWExplainResult(
							"x",
							core.MopValueType(1),
							core.MopValueType(2),
							0,
							0,
						)}, {WWExplainResult(
							"y",
							core.MopValueType(2),
							core.MopValueType(1),
							1,
							1,
						)}},
						Typ: "G0",
					},
				},
			},
			ImpossibleModels: nil,
			Not:              []string{"read-uncommitted"},
		}
		require.Equal(t, expect, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G0"},
		}, h))
	}
	// G1a
	if switches {
		t1 := mustParseOp(`{:type :fail, :value [[:append x 1]]}`)
		t2 := mustParseOp(`{:type :ok, :value [[:r x [1]]]}`)
		h := []core.Op{t1, t2}

		require.Equal(t, txn.CheckResult{
			IsUnknown:    true,
			AnomalyTypes: []string{"empty-transaction-graph"},
			Anomalies: core.Anomalies{
				"empty-transaction-graph": []core.Anomaly{},
			},
			Not: []string{"read-atomic", "read-committed"},
		}, check(txn.Opts{
			ConsistencyModels: []string{"read-uncommitted"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1a", "empty-transaction-graph"},
			Anomalies: core.Anomalies{
				"empty-transaction-graph": []core.Anomaly{},
				"G1a": []core.Anomaly{G1Conflict{
					Op: withIndex(t2, 1),
					Mop: core.Read(
						"x",
						[]int{1},
					),
					Writer:  withIndex(t1, 0),
					Element: 1,
				}},
			},
			Not: []string{"read-atomic", "read-committed"},
		}, check(txn.Opts{
			ConsistencyModels: []string{"read-committed"},
		}, h))

		require.Equal(t, txn.CheckResult{
			IsUnknown:    true,
			AnomalyTypes: []string{"empty-transaction-graph"},
			Anomalies: core.Anomalies{
				"empty-transaction-graph": []core.Anomaly{},
			},
			Not: []string{"read-atomic", "read-committed"},
		}, check(txn.Opts{
			ConsistencyModels: []string{"G2"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1a", "empty-transaction-graph"},
			Anomalies: core.Anomalies{
				"empty-transaction-graph": []core.Anomaly{},
				"G1a": []core.Anomaly{G1Conflict{
					Op: withIndex(t2, 1),
					Mop: core.Read(
						"x",
						[]int{1},
					),
					Writer:  withIndex(t1, 0),
					Element: 1,
				}},
			},
			Not: []string{"read-atomic", "read-committed"},
		}, check(txn.Opts{
			ConsistencyModels: []string{"repeatable-read"},
		}, h))
	}
	// G1b
	if switches {
		t1s := mustParseOp(`{:type :invoke, :value [[:append x 1] [:append x 2]] :index 0}`)
		t1e := mustParseOp(`{:type :ok, :value [[:append x 1] [:append x 2]] :index 1}`)
		t2s := mustParseOp(`{:type :invoke, :value [[:r x [1]]] :index 2}`)
		t2e := mustParseOp(`{:type :ok, :value [[:r x [1]]] :index 3}`)

		h := []core.Op{t1s, t1e, t2s, t2e}
		require.Equal(t, txn.CheckResult{Valid: true}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G0"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1b"},
			Anomalies: core.Anomalies{
				"G1b": []core.Anomaly{G1Conflict{
					Op:     t2e,
					Writer: t1e,
					Mop: core.Read(
						"x",
						[]int{1},
					),
					Element: 1,
				}},
			},
			Not: []string{"read-committed"},
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G1"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G2"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1b"},
			Anomalies: core.Anomalies{
				"G1b": []core.Anomaly{G1Conflict{
					Op:     t2e,
					Writer: t1e,
					Mop: core.Read(
						"x",
						[]int{1},
					),
					Element: 1,
				}},
			},
			Not: []string{"read-committed"},
		}, check(txn.Opts{
			ConsistencyModels: []string{"strict-serializable"},
		}, h))
	}
	// G1c
	if switches {
		t1 := mustParseOp(`{:type :ok, :value [[:append x 1] [:r y [1]]]}`)
		t2 := mustParseOp(`{:type :ok, :value [[:append x 2] [:append y 1]]}`)
		t3 := mustParseOp(`{:type :ok, :value [[:r x [1 2]] [:r y [1]]]}`)
		h := []core.Op{t1, t2, t3}

		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G0"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1c"},
			Anomalies: core.Anomalies{
				"G1c": []core.Anomaly{core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{withIndex(t2, 1), withIndex(t1, 0), withIndex(t2, 1)},
					},
					Steps: []core.Step{{WRExplainResult(
						"y",
						core.MopValueType(1),
						1,
						1,
					)}, {WWExplainResult(
						"x",
						core.MopValueType(1),
						core.MopValueType(2),
						0,
						0,
					)}},
					Typ: "G1c",
				}},
			},
			Not: []string{"read-committed"},
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G1"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G2"},
		}, h))
	}

	// G-single
	if switches {
		t1 := mustParseOp(`{:type :ok, :value [[:append x 1] [:append y 1]]}`)
		t2 := mustParseOp(`{:type :ok, :value [[:append x 2] [:r y nil]]}`)
		t3 := mustParseOp(`{:type :ok, :value [[:r x [1 2]]]}`)
		h := []core.Op{t1, t2, t3}

		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G0"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G1"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G-single"},
			Anomalies: core.Anomalies{
				"G-single": []core.Anomaly{core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{withIndex(t2, 1), withIndex(t1, 0), withIndex(t2, 1)},
					},
					Steps: []core.Step{{RWExplainResult(
						"y",
						initMagicNumber,
						core.MopValueType(1),
						1,
						1,
					)}, {WWExplainResult(
						"x",
						core.MopValueType(1),
						core.MopValueType(2),
						0,
						0,
					)}},
					Typ: "G-single",
				}},
			},
			Not: []string{"consistent-view"},
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G-single"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G-single"},
			Anomalies: core.Anomalies{
				"G-single": []core.Anomaly{core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{withIndex(t2, 1), withIndex(t1, 0), withIndex(t2, 1)},
					},
					Steps: []core.Step{{RWExplainResult(
						"y",
						initMagicNumber,
						core.MopValueType(1),
						1,
						1,
					)}, {WWExplainResult(
						"x",
						core.MopValueType(1),
						core.MopValueType(2),
						0,
						0,
					)}},
					Typ: "G-single",
				}},
			},
			Not: []string{"consistent-view"},
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G2"},
		}, h))
	}

	// G2
	if switches {
		t1 := mustParseOp(`{:type :ok, :value [[:append x 1] [:r y nil]]}`)
		t2 := mustParseOp(`{:type :ok, :value [[:append y 1] [:r x nil]]}`)
		h := []core.Op{t1, t2}

		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G0"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G1"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"read-committed"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G2-item"},
			Anomalies: core.Anomalies{
				"G2-item": []core.Anomaly{core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{withIndex(t1, 0), withIndex(t2, 1), withIndex(t1, 0)},
					},
					Steps: []core.Step{{RWExplainResult(
						"y",
						initMagicNumber,
						core.MopValueType(1),
						1,
						0,
					)}, {RWExplainResult(
						"x",
						initMagicNumber,
						core.MopValueType(1),
						1,
						0,
					)}},
					Typ: "G2-item",
				}},
			},
			Not: []string{"repeatable-read"},
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G2"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G2-item"},
			Anomalies: core.Anomalies{
				"G2-item": []core.Anomaly{core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{withIndex(t1, 0), withIndex(t2, 1), withIndex(t1, 0)},
					},
					Steps: []core.Step{{RWExplainResult(
						"y",
						initMagicNumber,
						core.MopValueType(1),
						1,
						0,
					)}, {RWExplainResult(
						"x",
						initMagicNumber,
						core.MopValueType(1),
						1,
						0,
					)}},
					Typ: "G2-item",
				}},
			},
			Not: []string{"repeatable-read"},
		}, check(txn.Opts{
			ConsistencyModels: []string{"serializable"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G2-item"},
			Anomalies: core.Anomalies{
				"G2-item": []core.Anomaly{core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{withIndex(t1, 0), withIndex(t2, 1), withIndex(t1, 0)},
					},
					Steps: []core.Step{{RWExplainResult(
						"y",
						initMagicNumber,
						core.MopValueType(1),
						1,
						0,
					)}, {RWExplainResult(
						"x",
						initMagicNumber,
						core.MopValueType(1),
						1,
						0,
					)}},
					Typ: "G2-item",
				}},
			},
			Not: []string{"repeatable-read"},
		}, check(txn.Opts{
			ConsistencyModels: []string{"repeatable-read"},
		}, h))
	}

	// Strict-1SR violation
	if switches {
		t0 := mustParseOp(`{:index 0, :type :invoke, :value [[:append x 1]]}`)
		t0p := mustParseOp(`{:index 1, :type :ok, :value [[:append x 1]]}`)
		t1 := mustParseOp(`{:index 2, :type :invoke, :value [[:append x 2]]}`)
		t1p := mustParseOp(`{:index 3, :type :ok, :value [[:append x 2]]} `)
		t2 := mustParseOp(`{:index 4, :type :invoke, :value [[:r x nil]]}`)
		t2p := mustParseOp(`{:index 5, :type :ok, :value [[:r x [1]]]}`)
		t3 := mustParseOp(`{:index 6, :type :invoke, :value [[:r x nil]]}`)
		t3p := mustParseOp(`{:index 7, :type :ok, :value [[:r x [1 2]]]}`)
		h := []core.Op{t0, t0p, t1, t1p, t2, t2p, t3, t3p}

		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G2"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{"serializable"},
		}, h))

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G-single-realtime"},
			Anomalies: core.Anomalies{
				"G-single-realtime": []core.Anomaly{core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{t2p, t1p, t2p},
					},
					Steps: []core.Step{{RWExplainResult(
						"x",
						core.MopValueType(1),
						core.MopValueType(2),
						0,
						0,
					)}, {core.RealtimeExplainResult{
						PreEnd:    t1p,
						PostStart: t2,
					}}},
					Typ: "G-single-realtime",
				}},
			},
			Not: []string{"strict-serializable"},
		}, check(txn.Opts{
			ConsistencyModels: []string{"strict-serializable"},
		}, h))
	}

	// contradictory read orders
	if switches {
		t1 := mustParseOp(`{:type :ok, :value [[:append x 1] [:r y [1]]]}`)
		t2 := mustParseOp(`{:type :ok, :value [[:append x 2]]}`)
		t3 := mustParseOp(`{:type :ok, :value [[:append x 3] [:append y 1]]}`)
		t4 := mustParseOp(`{:type :ok, :value [[:r x [1 3]]]}`)
		t5 := mustParseOp(`{:type :ok, :value [[:r x [1 2 3]]]}`)
		h := []core.Op{t1, t2, t3, t4, t5}

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1c", "incompatible-order"},
			Anomalies: core.Anomalies{
				"incompatible-order": []core.Anomaly{incompatibleOrder{
					Key:    "x",
					Values: [][]core.MopValueType{{1, 3}, {1, 2, 3}},
				}},
				"G1c": []core.Anomaly{core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{withIndex(t3, 2), withIndex(t1, 0), withIndex(t3, 2)},
					},
					Steps: []core.Step{{WRExplainResult(
						"y",
						core.MopValueType(1),
						1,
						1,
					)}, {WWExplainResult(
						"x",
						core.MopValueType(1),
						core.MopValueType(3),
						0,
						0,
					)}},
					Typ: "G1c",
				}},
			},
			Not: []string{"read-committed", "read-atomic"},
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G1"},
		}, h))
	}

	// dirty update
	if switches {
		if switches {
			t1 := mustParseOp(`{:process 0, :type :fail, :value [[:append x 1]]}`)
			h := []core.Op{t1}

			require.Equal(t, txn.CheckResult{
				IsUnknown:    true,
				AnomalyTypes: []string{"empty-transaction-graph"},
				Anomalies: core.Anomalies{
					"empty-transaction-graph": []core.Anomaly{},
				},
				Not: []string{},
			}, check(txn.Opts{
				ConsistencyModels: []string{},
				Anomalies:         []string{"dirty-update"},
			}, h))
		}
		if switches {
			t1 := mustParseOp(`{:process 0, :type :fail, :value [[:append x 1]]}`)
			t2 := mustParseOp(`{:process 1, :type :ok, :value [[:append x 2]]}`)
			t3 := mustParseOp(`{:process 2, :type :ok, :value [[:r x [1 2]]]}`)
			h := []core.Op{t1, t2, t3}

			require.Equal(t, txn.CheckResult{
				Valid:        false,
				AnomalyTypes: []string{"dirty-update"},
				Anomalies: core.Anomalies{
					"dirty-update": []core.Anomaly{DirtyUpdateConflict{
						Key:    "x",
						Values: []core.MopValueType{1, 2},
						Op1:    withIndex(t1, 0),
						Op2:    withIndex(t2, 1),
					}},
				},
				Not: []string{"read-atomic", "read-committed"},
			}, check(txn.Opts{
				ConsistencyModels: []string{},
				Anomalies:         []string{"dirty-update"},
			}, h))
		}
		if switches {
			t1 := mustParseOp(`{:process 0, :type :fail, :value [[:append x 1]]}`)
			t2 := mustParseOp(`{:process 1, :type :info, :value [[:append x 2]]}`)
			t3 := mustParseOp(`{:process 2, :type :ok, :value [[:append x 3]]}`)
			t4 := mustParseOp(`{:process 3, :type :ok, :value [[:r x [1 2 3]]]}`)
			h := []core.Op{t1, t2, t3, t4}

			require.Equal(t, txn.CheckResult{
				Valid:        false,
				AnomalyTypes: []string{"dirty-update"},
				Anomalies: core.Anomalies{
					"dirty-update": []core.Anomaly{DirtyUpdateConflict{
						Key:    "x",
						Values: []core.MopValueType{1, 2, 3},
						Op1:    withIndex(t1, 0),
						Op2:    withIndex(t3, 2),
					}},
				},
				Not: []string{"read-atomic", "read-committed"},
			}, check(txn.Opts{
				ConsistencyModels: []string{},
				Anomalies:         []string{"dirty-update"},
			}, h))
		}
	}
	if switches {
		t1 := mustParseOp(`{:type :ok, :value [[:append x 1] [:r y [1]]]}`)
		t2 := mustParseOp(`{:type :ok, :value [[:append x 2] [:append y 1]]}`)
		t3 := mustParseOp(`{:type :ok, :value [[:r x [1 2 1]]]}`)
		h := []core.Op{t1, t2, t3}

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1c", "duplicate-elements"},
			Anomalies: core.Anomalies{
				"duplicate-elements": []core.Anomaly{duplicateConflict{
					Op:  withIndex(t3, 2),
					Mop: core.Read("x", []int{1, 2, 1}),
					Dups: map[core.MopValueType]int{
						1: 2,
					},
				}},
				"G1c": []core.Anomaly{core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{withIndex(t2, 1), withIndex(t1, 0), withIndex(t2, 1)},
					},
					Steps: []core.Step{{WRExplainResult(
						"y",
						core.MopValueType(1),
						1,
						1,
					)}, {WWExplainResult(
						"x",
						core.MopValueType(1),
						core.MopValueType(2),
						0,
						0,
					)}},
					Typ: "G1c",
				}},
			},
			Not: []string{"read-uncommitted"},
		}, check(txn.Opts{
			ConsistencyModels: []string{"read-committed"},
		}, h))
	}
	if switches {
		t1 := mustParseOp(`{:type :ok, :value [[:append x 1] [:append x 2] [:append x 4]]}`)
		t2 := mustParseOp(`{:type :ok, :value [[:append x 3] [:r x [1 2 3 4]]]}`)
		h := []core.Op{t1, t2}

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"internal"},
			Anomalies: core.Anomalies{
				"internal": []core.Anomaly{InternalConflict{
					Op: withIndex(t2, 1),
					Mop: core.Read(
						"x",
						[]int{1, 2, 3, 4},
					),
					Expected: []int{unknownPrefixMagicNumber, 3},
				}},
			},
			Not: []string{"read-uncommitted", "read-atomic"},
		}, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"internal"},
		}, h))
	}
}

func TestRepeatableRead(t *testing.T) {
	t1 := mustParseOp(`{:type :ok, :value [[:r x nil] [:append y 1]]}`)
	t2 := mustParseOp(`{:type :ok, :value [[:r y nil] [:append x 1]]}`)
	h := []core.Op{t1, t2}

	require.Equal(t, txn.CheckResult{
		Valid:        false,
		AnomalyTypes: []string{"G2-item"},
		Anomalies: core.Anomalies{
			"G2-item": []core.Anomaly{
				core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{withIndex(t1, 0), withIndex(t2, 1), withIndex(t1, 0)},
					},
					Steps: []core.Step{{RWExplainResult(
						"x",
						initMagicNumber,
						1,
						0,
						1,
					)}, {RWExplainResult(
						"y",
						initMagicNumber,
						core.MopValueType(1),
						0,
						1,
					)}},
					Typ: "G2-item",
				},
			},
		},
		Not: []string{"repeatable-read"},
	}, check(txn.Opts{
		ConsistencyModels: []string{"repeatable-read"},
	}, h))
}

func TestGNonadjacent(t *testing.T) {
	t1 := mustParseOp(`{:index 0, :type :invoke, :value [[:append x 1]]}`)
	t1p := mustParseOp(`{:index 1, :type :ok, :value [[:append x 1]]}`)
	t2 := mustParseOp(`{:index 2, :type :invoke, :value [[:r x [1]] [:r y nil]]}`)
	t2p := mustParseOp(`{:index 3, :type :ok, :value [[:r x [1]] [:r y nil]]}`)
	t3 := mustParseOp(`{:index 4, :type :invoke, :value [[:append y 1]]}`)
	t3p := mustParseOp(`{:index 5, :type :ok, :value [[:append y 1]]}`)
	t4 := mustParseOp(`{:index 6, :type :invoke, :value [[:r y [1]] [:r x nil]]}`)
	t4p := mustParseOp(`{:index 7, :type :ok, :value [[:r y [1]] [:r x nil]]}`)
	h := []core.Op{t1, t1p, t2, t2p, t3, t3p, t4, t4p}

	got := check(txn.Opts{}, h)

	require.Equal(t, txn.CheckResult{
		Valid:        false,
		AnomalyTypes: []string{"G-nonadjacent"},
		Anomalies: core.Anomalies{
			"G-nonadjacent": []core.Anomaly{
				core.CycleExplainerResult{
					Circle: core.Circle{
						Path: []core.Op{t1p, t2p, t3p, t4p, t1p},
					},
					Steps: []core.Step{{WRExplainResult(
						"x",
						1,
						0,
						0,
					)}, {RWExplainResult(
						"y",
						initMagicNumber,
						core.MopValueType(1),
						1,
						0,
					)}, {WRExplainResult(
						"y",
						core.MopValueType(1),
						0,
						0,
					)}, {RWExplainResult(
						"x",
						initMagicNumber,
						core.MopValueType(1),
						1,
						0,
					)}},
					Typ: "G-nonadjacent",
				},
			},
		},
		Not: []string{"strong-session-snapshot-isolation", "serializable"},
	}, got)

}

func TestCheck(t *testing.T) {
	var history = core.History{
		core.Op{Type: core.OpTypeOk,
			Value: &[]core.Mop{
				core.Append(
					"x",
					1,
				),
				core.Read(
					"y",
					[]int{1},
				),
			}},
		core.Op{Type: core.OpTypeOk,
			Value: &[]core.Mop{
				core.Append(
					"x",
					2,
				),
				core.Append(
					"y",
					1,
				)}},
		core.Op{Type: core.OpTypeOk,
			Value: &[]core.Mop{
				core.Read(
					"x",
					[]int{1, 2},
				)}},
	}

	result := Check(txn.Opts{
		ConsistencyModels: []core.ConsistencyModelName{"serializable"},
	}, history)

	_ = result
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
	// the shortest path we found isn't belong to any anomalies...
	result := Check(txn.Opts{}, history)
	_ = result
}

func check(opts txn.Opts, h core.History) txn.CheckResult {
	result := Check(opts, h)
	result.AlsoNot = nil
	return result
}

func mustParseOp(opString string) core.Op {
	op, err := core.ParseOp(opString)
	if err != nil {
		log.Fatalf("expect no error, got %v", err)
	}
	return op
}

func withIndex(op core.Op, index int) core.Op {
	o := op
	o.Index.Set(index)
	return o
}
