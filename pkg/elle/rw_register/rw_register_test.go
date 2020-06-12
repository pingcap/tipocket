package rwregister

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

func TestInternalCase(t *testing.T) {
	op0 := MustParseOp("wx1rx1")
	op1 := MustParseOp("ry2wx2rx2wx1rx1")
	op2 := MustParseOp("rx1wx2rx1")

	require.Equal(t, nil, internalOp(op0))
	require.Equal(t, nil, internalOp(op1))

	mop := (*op2.Value)[2]
	expectedMop := mop.Copy()
	expectedMop.M["x"] = NewInt(2)
	conflict := InternalConflict{
		Op:       op2,
		Mop:      mop,
		Expected: expectedMop,
	}
	require.Equal(t, conflict, internalOp(op2))

	require.Equal(t, GCaseTp{conflict}, internal(core.History{op0, op1, op2}))
}

func TestG1ACases(t *testing.T) {
	good := core.History{
		MustParseOp("wx1"),
		MustParseOp("rx1ry_"),
	}
	bad := core.History{
		MustParseOp("wx2").WithType(core.OpTypeFail),
		MustParseOp("rx2"),
	}

	conflict := G1Conflict{
		Op:      bad[1],
		Mop:     (*bad[1].Value)[0],
		Writer:  bad[0],
		Element: "x",
	}

	require.Equal(t, GCaseTp(nil), g1aCases(core.History{}))
	require.Equal(t, GCaseTp(nil), g1aCases(good))
	require.Equal(t, GCaseTp{conflict}, g1aCases(bad))
}

func TestG1BCases(t *testing.T) {
	good := core.History{
		MustParseOp("wx1wx2"),
		MustParseOp("rx_rx2"),
	}
	bad := core.History{
		MustParseOp("wx2wx1"),
		MustParseOp("rx2"),
	}

	conflict := G1Conflict{
		Op:      bad[1],
		Mop:     (*bad[1].Value)[0],
		Writer:  bad[0],
		Element: "x",
	}

	require.Equal(t, GCaseTp(nil), g1bCases(core.History{}))
	require.Equal(t, GCaseTp(nil), g1bCases(good))
	require.Equal(t, GCaseTp{conflict}, g1bCases(bad))
}

func TestWRGraph(t *testing.T) {
	case1 := core.History{
		MustParseOp("wx1"),
		MustParseOp("rx1"),
		MustParseOp("wx2"),
		MustParseOp("rx2"),
	}
	expect1 := core.NewDirectedGraph()
	expect1.Link(core.Vertex{Value: case1[0]}, core.Vertex{Value: case1[1]}, core.WR)
	expect1.Link(core.Vertex{Value: case1[2]}, core.Vertex{Value: case1[3]}, core.WR)
	_, g1, _ := wrGraph(case1)
	require.Equal(t, expect1, g1)

	// write and read
	case2 := core.History{
		MustParseOp("wx0"),
		MustParseOp("rx0"),
	}
	expect2 := core.NewDirectedGraph()
	expect2.Link(core.Vertex{Value: case2[0]}, core.Vertex{Value: case2[1]}, core.WR)
	_, g2, _ := wrGraph(case2)
	require.Equal(t, expect2, g2)

	// chain on one register
	case3 := core.History{
		MustParseOp("rx0wx1"),
		MustParseOp("rx1wx0"),
	}
	expect3 := core.NewDirectedGraph()
	expect3.Link(core.Vertex{Value: case3[0]}, core.Vertex{Value: case3[1]}, core.WR)
	expect3.Link(core.Vertex{Value: case3[1]}, core.Vertex{Value: case3[0]}, core.WR)
	_, g3, _ := wrGraph(case3)
	require.Equal(t, expect3, g3)

	// chain across two registers
	case4 := core.History{
		MustParseOp("rx0wy1"),
		MustParseOp("ry1wx0"),
	}
	expect4 := core.NewDirectedGraph()
	expect4.Link(core.Vertex{Value: case4[0]}, core.Vertex{Value: case4[1]}, core.WR)
	expect4.Link(core.Vertex{Value: case4[1]}, core.Vertex{Value: case4[0]}, core.WR)
	_, g4, _ := wrGraph(case4)
	require.Equal(t, expect4, g4)

	// write skew
	case5 := core.History{
		MustParseOp("rx0ry0wx1"),
		MustParseOp("rx0ry0wy1"),
	}
	expect5 := core.NewDirectedGraph()
	_, g5, _ := wrGraph(case5)
	require.Equal(t, expect5, g5)
}

func TestGetKeys(t *testing.T) {
	require.Equal(t, getKeys(MustParseOp("wx1")), []string{"x"})
	require.Equal(t, getKeys(MustParseOp("wx1wy2wz3")), []string{"x", "y", "z"})
	require.Equal(t, getKeys(MustParseOp("wx1wy2rz3").WithType(core.OpTypeInfo)), []string{"x", "y"})
}

func TestGetVersion(t *testing.T) {
	require.Equal(t, getVersion("x", MustParseOp("wx1wx2wx3")), NewInt(3))
	require.Equal(t, getVersion("x", MustParseOp("wx1wx2wx3wy1wy2")), NewInt(3))
	require.Equal(t, getVersion("y", MustParseOp("wx1wx2wx3wy1wy2")), NewInt(2))
	require.Equal(t, getVersion("x", MustParseOp("rx1rx2wx3wx4wx5wy1wy2ry2ry3")), NewInt(1))
	require.Equal(t, getVersion("y", MustParseOp("rx1rx2wx3wx4wx5wy1wy2ry3")), NewInt(2))
	require.Equal(t, getVersion("y", MustParseOp("rx1rx2wx3wx4wx5ry1wy2wy3")), NewInt(1))
}

func TestVersionGraph(t *testing.T) {
	// r->w
	history1 := core.History{
		MustParseOp("rx1"),
		MustParseOp("wx2"),
	}
	expect1x := core.NewDirectedGraph()
	expect1x.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(2)}, core.Realtime)
	expect1 := map[string]*core.DirectedGraph{
		"x": expect1x,
	}
	g1 := core.NewDirectedGraph()
	g1.Link(core.Vertex{Value: history1[0]}, core.Vertex{Value: history1[1]}, core.Realtime)
	r1 := transactionGraph2VersionGraphs(core.Realtime, history1, g1)
	require.Equal(t, expect1, r1)

	// fork-join
	history2 := core.History{
		MustParseOp("rx1"),
		MustParseOp("wx2"),
		MustParseOp("wx3"),
		MustParseOp("rx4"),
	}
	expect2x := core.NewDirectedGraph()
	expect2x.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(2)}, core.Realtime)
	expect2x.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(3)}, core.Realtime)
	expect2x.Link(core.Vertex{Value: NewInt(2)}, core.Vertex{Value: NewInt(4)}, core.Realtime)
	expect2x.Link(core.Vertex{Value: NewInt(3)}, core.Vertex{Value: NewInt(4)}, core.Realtime)
	expect2 := map[string]*core.DirectedGraph{
		"x": expect2x,
	}
	g2 := core.NewDirectedGraph()
	g2.Link(core.Vertex{Value: history2[0]}, core.Vertex{Value: history2[1]}, core.Realtime)
	g2.Link(core.Vertex{Value: history2[0]}, core.Vertex{Value: history2[2]}, core.Realtime)
	g2.Link(core.Vertex{Value: history2[1]}, core.Vertex{Value: history2[3]}, core.Realtime)
	g2.Link(core.Vertex{Value: history2[2]}, core.Vertex{Value: history2[3]}, core.Realtime)
	r2 := transactionGraph2VersionGraphs(core.Realtime, history2, g2)
	require.Equal(t, expect2, r2)

	// external ww
	history3 := core.History{
		MustParseOp("wx1wx2"),
		MustParseOp("wx3wx4rx5"),
	}
	expect3x := core.NewDirectedGraph()
	expect3x.Link(core.Vertex{Value: NewInt(2)}, core.Vertex{Value: NewInt(4)}, core.Realtime)
	expect3 := map[string]*core.DirectedGraph{
		"x": expect3x,
	}
	g3 := core.NewDirectedGraph()
	g3.Link(core.Vertex{Value: history3[0]}, core.Vertex{Value: history3[1]}, core.Realtime)
	r3 := transactionGraph2VersionGraphs(core.Realtime, history3, g3)
	require.Equal(t, expect3, r3)

	// external wr
	history4 := core.History{
		MustParseOp("wx1wx2"),
		MustParseOp("rx3rx4wx5"),
	}
	expect4x := core.NewDirectedGraph()
	expect4x.Link(core.Vertex{Value: NewInt(2)}, core.Vertex{Value: NewInt(3)}, core.Realtime)
	expect4 := map[string]*core.DirectedGraph{
		"x": expect4x,
	}
	g4 := core.NewDirectedGraph()
	g4.Link(core.Vertex{Value: history4[0]}, core.Vertex{Value: history4[1]}, core.Realtime)
	r4 := transactionGraph2VersionGraphs(core.Realtime, history4, g4)
	require.Equal(t, expect4, r4)

	// external rw
	history5 := core.History{
		MustParseOp("rx1rx2"),
		MustParseOp("wx3wx4rx5"),
	}
	expect5x := core.NewDirectedGraph()
	expect5x.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(4)}, core.Realtime)
	expect5 := map[string]*core.DirectedGraph{
		"x": expect5x,
	}
	g5 := core.NewDirectedGraph()
	g5.Link(core.Vertex{Value: history5[0]}, core.Vertex{Value: history5[1]}, core.Realtime)
	r5 := transactionGraph2VersionGraphs(core.Realtime, history5, g5)
	require.Equal(t, expect5, r5)

	// external rr
	history6 := core.History{
		MustParseOp("rx1rx2"),
		MustParseOp("rx3rx4wx5"),
	}
	expect6x := core.NewDirectedGraph()
	expect6x.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(3)}, core.Realtime)
	expect6 := map[string]*core.DirectedGraph{
		"x": expect6x,
	}
	g6 := core.NewDirectedGraph()
	g6.Link(core.Vertex{Value: history6[0]}, core.Vertex{Value: history6[1]}, core.Realtime)
	r6 := transactionGraph2VersionGraphs(core.Realtime, history6, g6)
	require.Equal(t, expect6, r6)

	// don't infer v1 -> v1 deps
	history7 := core.History{
		MustParseOp("wx1"),
		MustParseOp("rx1"),
	}
	expect7 := map[string]*core.DirectedGraph{
		"x": core.NewDirectedGraph(),
	}
	g7 := core.NewDirectedGraph()
	g7.Link(core.Vertex{Value: history7[0]}, core.Vertex{Value: history7[1]}, core.Realtime)
	r7 := transactionGraph2VersionGraphs(core.Realtime, history7, g7)
	require.Equal(t, expect7, r7)

	// don't infer deps on failed or crashed reads
	history8 := core.History{
		MustParseOp("wx1"),
		MustParseOp("rx2").WithType(core.OpTypeFail),
		MustParseOp("rx3").WithType(core.OpTypeInfo),
		MustParseOp("rx4").WithType(core.OpTypeFail),
		MustParseOp("rx5"),
		MustParseOp("rx6").WithType(core.OpTypeInfo),
		MustParseOp("rx7"),
	}
	expect8 := map[string]*core.DirectedGraph{
		"x": core.NewDirectedGraph(),
	}
	g8 := core.NewDirectedGraph()
	g8.Link(core.Vertex{Value: history8[0]}, core.Vertex{Value: history8[1]}, core.Realtime)
	g8.Link(core.Vertex{Value: history8[0]}, core.Vertex{Value: history8[2]}, core.Realtime)
	g8.Link(core.Vertex{Value: history8[3]}, core.Vertex{Value: history8[4]}, core.Realtime)
	g8.Link(core.Vertex{Value: history8[5]}, core.Vertex{Value: history8[6]}, core.Realtime)
	r8 := transactionGraph2VersionGraphs(core.Realtime, history8, g8)
	require.Equal(t, expect8, r8)

	// don't infer deps on failed writes, but do infer crashed
	history9 := core.History{
		MustParseOp("wx1"),
		MustParseOp("wx2").WithType(core.OpTypeFail),
		MustParseOp("rx3wx4").WithType(core.OpTypeInfo),
		MustParseOp("wx5").WithType(core.OpTypeFail),
		MustParseOp("rx6"),
		MustParseOp("wx8").WithType(core.OpTypeInfo),
		MustParseOp("rx9"),
	}
	expect9x := core.NewDirectedGraph()
	expect9x.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(4)}, core.Realtime)
	expect9x.Link(core.Vertex{Value: NewInt(8)}, core.Vertex{Value: NewInt(9)}, core.Realtime)
	expect9 := map[string]*core.DirectedGraph{
		"x": expect9x,
	}
	g9 := core.NewDirectedGraph()
	g9.Link(core.Vertex{Value: history9[0]}, core.Vertex{Value: history9[1]}, core.Realtime)
	g9.Link(core.Vertex{Value: history9[0]}, core.Vertex{Value: history9[2]}, core.Realtime)
	g9.Link(core.Vertex{Value: history9[3]}, core.Vertex{Value: history9[4]}, core.Realtime)
	g9.Link(core.Vertex{Value: history9[5]}, core.Vertex{Value: history9[6]}, core.Realtime)
	r9 := transactionGraph2VersionGraphs(core.Realtime, history9, g9)
	require.Equal(t, expect9, r9)

	// see through failed/crashed ops
	history10 := core.History{
		MustParseOp("wx1"),
		MustParseOp("rx_").WithType(core.OpTypeInfo),
		MustParseOp("rx3"),
		MustParseOp("wy1"),
		MustParseOp("wy2").WithType(core.OpTypeFail),
		MustParseOp("ry3"),
	}
	expect10x := core.NewDirectedGraph()
	expect10y := core.NewDirectedGraph()
	expect10x.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(3)}, core.Realtime)
	expect10y.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(3)}, core.Realtime)
	expect10 := map[string]*core.DirectedGraph{
		"x": expect10x,
		"y": expect10y,
	}
	g10 := core.NewDirectedGraph()
	g10.LinkToAll(core.Vertex{Value: history10[0]}, []core.Vertex{
		{Value: history10[1]},
		{Value: history10[2]},
	}, core.Realtime)
	g10.LinkToAll(core.Vertex{Value: history10[3]}, []core.Vertex{
		{Value: history10[4]},
		{Value: history10[5]},
	}, core.Realtime)
	r10 := transactionGraph2VersionGraphs(core.Realtime, history10, g10)
	require.Equal(t, expect10, r10)

	// see through seq. failed/crashed ops
	history11 := core.History{
		MustParseOp("wx1"),
		MustParseOp("rx_").WithType(core.OpTypeInfo),
		MustParseOp("wx2").WithType(core.OpTypeFail),
		MustParseOp("wx3"),
	}
	expect11x := core.NewDirectedGraph()
	expect11x.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(3)}, core.Realtime)
	expect11 := map[string]*core.DirectedGraph{
		"x": expect11x,
	}
	g11 := core.NewDirectedGraph()
	g11.Link(core.Vertex{Value: history11[0]}, core.Vertex{Value: history11[1]}, core.Realtime)
	g11.Link(core.Vertex{Value: history11[1]}, core.Vertex{Value: history11[2]}, core.Realtime)
	g11.Link(core.Vertex{Value: history11[2]}, core.Vertex{Value: history11[3]}, core.Realtime)
	r11 := transactionGraph2VersionGraphs(core.Realtime, history11, g11)
	require.Equal(t, expect11, r11)

	// see through other keys
	history12 := core.History{
		MustParseOp("wx1"),
		MustParseOp("wx2"),
		MustParseOp("wy2"),
		MustParseOp("wx3"),
	}
	expect12x := core.NewDirectedGraph()
	expect12x.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(2)}, core.Realtime)
	expect12x.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(3)}, core.Realtime)
	expect12 := map[string]*core.DirectedGraph{
		"x": expect12x,
		"y": core.NewDirectedGraph(),
	}
	g12 := core.NewDirectedGraph()
	g12.Link(core.Vertex{Value: history12[0]}, core.Vertex{Value: history12[1]}, core.Realtime)
	g12.Link(core.Vertex{Value: history12[0]}, core.Vertex{Value: history12[2]}, core.Realtime)
	g12.Link(core.Vertex{Value: history12[2]}, core.Vertex{Value: history12[3]}, core.Realtime)
	r12 := transactionGraph2VersionGraphs(core.Realtime, history12, g12)
	fmt.Println(r12)
	require.Equal(t, expect12, r12)
}

func TestVersionGraphs2TransactionGraph(t *testing.T) {
	// empty
	versions1 := map[string]*core.DirectedGraph{
		"x": core.NewDirectedGraph(),
	}
	var history1 core.History
	expect1 := core.NewDirectedGraph()
	require.Equal(t, expect1, versionGraphs2TransactionGraph(history1, versions1))

	// rr
	version2 := core.NewDirectedGraph()
	version2.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(2)}, core.ExtKey)
	version2.Link(core.Vertex{Value: NewInt(2)}, core.Vertex{Value: NewInt(3)}, core.ExtKey)
	versions2 := map[string]*core.DirectedGraph{
		"x": version2,
	}
	history2 := core.History{
		MustParseOp("rx1"),
		MustParseOp("rx2"),
	}
	expect2 := core.NewDirectedGraph()
	require.Equal(t, expect2, versionGraphs2TransactionGraph(history2, versions2))

	// wr
	version3 := core.NewDirectedGraph()
	version3.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(2)}, core.ExtKey)
	version3.Link(core.Vertex{Value: NewInt(2)}, core.Vertex{Value: NewInt(3)}, core.ExtKey)
	versions3 := map[string]*core.DirectedGraph{
		"x": version3,
	}
	history3 := core.History{
		MustParseOp("wx1"),
		MustParseOp("rx1"),
	}
	expect3 := core.NewDirectedGraph()
	require.Equal(t, expect3, versionGraphs2TransactionGraph(history3, versions3))

	// ww
	version4 := core.NewDirectedGraph()
	version4.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(2)}, core.ExtKey)
	version4.Link(core.Vertex{Value: NewInt(2)}, core.Vertex{Value: NewInt(3)}, core.ExtKey)
	versions4 := map[string]*core.DirectedGraph{
		"x": version4,
	}
	history4 := core.History{
		MustParseOp("wx1"),
		MustParseOp("wx2"),
	}
	expect4 := core.NewDirectedGraph()
	expect4.Link(core.Vertex{Value: history4[0]}, core.Vertex{Value: history4[1]}, core.WW)
	require.Equal(t, expect4, versionGraphs2TransactionGraph(history4, versions4))

	// rw
	version5 := core.NewDirectedGraph()
	version5.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(2)}, core.ExtKey)
	version5.Link(core.Vertex{Value: NewInt(2)}, core.Vertex{Value: NewInt(3)}, core.ExtKey)
	versions5 := map[string]*core.DirectedGraph{
		"x": version5,
	}
	history5 := core.History{
		MustParseOp("rx1"),
		MustParseOp("wx2"),
	}
	expect5 := core.NewDirectedGraph()
	expect5.Link(core.Vertex{Value: history5[0]}, core.Vertex{Value: history5[1]}, core.RW)
	require.Equal(t, expect5, versionGraphs2TransactionGraph(history5, versions5))

	// ignores internal writes/reads
	version6 := core.NewDirectedGraph()
	version6.Link(core.Vertex{Value: NewInt(1)}, core.Vertex{Value: NewInt(2)}, core.ExtKey)
	version6.Link(core.Vertex{Value: NewInt(2)}, core.Vertex{Value: NewInt(3)}, core.ExtKey)
	versions6 := map[string]*core.DirectedGraph{
		"x": version6,
	}
	history6 := core.History{
		MustParseOp("wx1wx2"),
		MustParseOp("rx2rx3"),
	}
	expect6 := core.NewDirectedGraph()
	require.Equal(t, expect6, versionGraphs2TransactionGraph(history6, versions6))
}

func TestChecker(t *testing.T) {
	switches := true
	// G0
	if switches {
		var (
			t1, t1Ok = Pair(MustParseOp("wx1wy2").WithProcess(0))
			t2, t2Ok = Pair(MustParseOp("wx2wy1").WithProcess(1))
			t3, t3Ok = Pair(MustParseOp("rx2").WithProcess(0))
			t4, t4Ok = Pair(MustParseOp("ry2").WithProcess(1))
			optG0    = GraphOption{
				SequentialKeys: true,
			}
		)

		rs1 := check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G0"},
		}, core.History{t1, t2}, optG0)
		require.Equal(t, txn.CheckResult{
			IsUnknown:    true,
			AnomalyTypes: []string{"empty-transaction-graph"},
			Anomalies: core.Anomalies{
				"empty-transaction-graph": []core.Anomaly{},
			},
			Not: []string{},
		}, rs1)

		rs2 := check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G0"},
		}, core.History{t1, t1Ok, t2, t2Ok, t3, t3Ok, t4, t4Ok}, optG0)

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G0"},
			Anomalies: core.Anomalies{
				"G0": []core.Anomaly{
					core.CycleExplainerResult{
						Circle: core.Circle{
							Path: []core.Op{withIndex(t1Ok, 1), withIndex(t2Ok, 3), withIndex(t1Ok, 1)},
						},
						Steps: []core.Step{
							{
								Result: WWExplainResult("x", core.MopValueType(NewInt(1)), core.MopValueType(NewInt(2))),
							},
							{
								Result: WWExplainResult("y", core.MopValueType(NewInt(1)), core.MopValueType(NewInt(2))),
							},
						},
						Typ: "G0",
					},
				},
			},
			Not: []string{"read-uncommitted"},
		}, rs2)
	}
	// G1a
	if switches {
		t1 := MustParseOp("wx1").WithType(core.OpTypeFail)
		t2 := MustParseOp("rx1")

		rs1 := check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G1"},
		}, core.History{t2, t1}, GraphOption{})

		require.Equal(t, txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1a", "empty-transaction-graph"},
			Anomalies: core.Anomalies{
				"G1a": []core.Anomaly{
					G1Conflict{
						Op:      withIndex(t2, 0),
						Mop:     (*t2.Value)[0],
						Writer:  withIndex(t1, 1),
						Element: "x",
					},
				},
				"empty-transaction-graph": []core.Anomaly{},
			},
			Not: []string{"read-atomic", "read-committed"},
		}, rs1)
	}
	// G1b
	if switches {
		var (
			t1      = MustParseOp("wx1wx2")
			t2      = MustParseOp("rx1")
			history = core.History{t1, t2}
		)

		rs1 := check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G0"},
		}, history, GraphOption{})
		require.Equal(t, txn.CheckResult{
			IsUnknown:    true,
			Valid:        false,
			AnomalyTypes: []string{"empty-transaction-graph"},
			Anomalies: core.Anomalies{
				"empty-transaction-graph": []core.Anomaly{},
			},
			Not: []string{"read-committed"},
		}, rs1)

		expect23 := txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1b", "empty-transaction-graph"},
			Anomalies: core.Anomalies{
				"empty-transaction-graph": []core.Anomaly{},
				"G1b": []core.Anomaly{
					G1Conflict{
						Op:      t2.WithIndex(1),
						Mop:     (*t2.Value)[0],
						Writer:  t1.WithIndex(0),
						Element: "x",
					},
				},
			},
			Not: []string{"read-committed"},
		}

		rs2 := check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G1"},
		}, history, GraphOption{})

		rs3 := check(txn.Opts{
			ConsistencyModels: []string{"repeatable-read"},
			Anomalies:         []string{},
		}, history, GraphOption{})

		require.Equal(t, expect23, rs2)
		require.Equal(t, expect23, rs3)
	}
	// G1c
	if switches {
		var (
			t1      = MustParseOp("wx1ry1")
			t2      = MustParseOp("wy1rx1")
			history = core.History{t1, t2}
			msg     = core.CycleExplainerResult{
				Circle: core.Circle{
					Path: []core.PathType{
						t1.WithIndex(0),
						t2.WithIndex(1),
						t1.WithIndex(0),
					},
				},
				Steps: []core.Step{
					{
						Result: WRExplainResult("x", core.MopValueType(NewInt(1))),
					},
					{
						Result: WRExplainResult("y", core.MopValueType(NewInt(1))),
					},
				},
				Typ: "G1c",
			}
		)

		// G0/read-uncommitted won't see this
		expectG0 := txn.CheckResult{
			Valid: true,
		}
		require.Equal(t, expectG0, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G0"},
		}, history, GraphOption{}))
		require.Equal(t, expectG0, check(txn.Opts{
			ConsistencyModels: []string{"read-uncommitted"},
			Anomalies:         []string{},
		}, history, GraphOption{}))

		// But G1 will, as will a read-committed checker.
		expectG1 := txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1c"},
			Anomalies: core.Anomalies{
				"G1c": []core.Anomaly{msg},
			},
			Not: []string{"read-committed"},
		}
		require.Equal(t, expectG1, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G1c"},
		}, history, GraphOption{}))
		require.Equal(t, expectG1, check(txn.Opts{
			ConsistencyModels: []string{"read-committed"},
			Anomalies:         []string{},
		}, history, GraphOption{}))

		// A checker looking for G2 alone won't care about this,
		// because G1c is not G2
		expectG2 := txn.CheckResult{
			Valid: true,
		}
		require.Equal(t, expectG2, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G2"},
		}, history, GraphOption{}))

		// But a serializable checker will catch this,
		// because serializability proscribes both G1 and G2.
		expectSerializable := txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G1c"},
			Anomalies: core.Anomalies{
				"G1c": []core.Anomaly{msg},
			},
			Not: []string{"read-committed"},
		}
		require.Equal(t, expectSerializable, check(txn.Opts{
			ConsistencyModels: []string{"serializable"},
			Anomalies:         []string{},
		}, history, GraphOption{}))
	}
	// G2-item
	if switches {
		var (
			t1, t1Ok = Pair(MustParseOp("rx1ry1").WithProcess(0))
			t2, t2Ok = Pair(MustParseOp("rx1wy2").WithProcess(1))
			t3, t3Ok = Pair(MustParseOp("ry1wx2").WithProcess(2))
			history  = core.History{t1, t1Ok, t2, t3, t3Ok, t2Ok}
		)

		// Read committed won't see this, since it's G2-item.
		require.Equal(t, txn.CheckResult{
			Valid: true,
		}, check(txn.Opts{
			ConsistencyModels: []string{"read-committed"},
			Anomalies:         []string{},
		}, history, GraphOption{
			LinearizableKeys: true,
		}))

		// But repeatable read will!
		expectRR := txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G2-item"},
			Anomalies: core.Anomalies{
				"G2-item": []core.Anomaly{
					core.CycleExplainerResult{
						Circle: core.Circle{
							Path: []core.PathType{
								t3Ok.WithIndex(4),
								t2Ok.WithIndex(5),
								t3Ok.WithIndex(4),
							},
						},
						Steps: []core.Step{
							{
								Result: RWExplainResult("y", core.MopValueType(NewInt(1)), core.MopValueType(NewInt(2))),
							},
							{
								Result: RWExplainResult("x", core.MopValueType(NewInt(1)), core.MopValueType(NewInt(2))),
							},
						},
						Typ: "G2-item",
					},
				},
			},
			Not: []string{"repeatable-read"},
		}
		require.Equal(t, expectRR, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"G2"},
		}, history, GraphOption{
			LinearizableKeys: true,
		}))
	}
	// internal
	if switches {
		t1 := MustParseOp("rx1rx2")
		history := core.History{t1}
		expect := txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"empty-transaction-graph", "internal"},
			Anomalies: core.Anomalies{
				"internal": []core.Anomaly{
					InternalConflict{
						Op:       t1.WithIndex(0),
						Mop:      (*t1.Value)[1],
						Expected: (*t1.Value)[0],
					},
				},
				"empty-transaction-graph": []core.Anomaly{},
			},
			Not: []string{"read-atomic"},
		}
		require.Equal(t, expect, check(txn.Opts{
			ConsistencyModels: []string{},
			Anomalies:         []string{"internal"},
		}, history, GraphOption{}))
	}
	// initial state
	if switches {
		var (
			t1, t1Ok = Pair(MustParseOp("rx_ry1").WithProcess(0))
			t2, t2Ok = Pair(MustParseOp("wy1wx2").WithProcess(0))
			history  = core.History{t1, t2, t2Ok, t1Ok}
		)
		history.AttachIndexIfNoExists()
		expect := txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G-single"},
			Anomalies: core.Anomalies{
				"G-single": []core.Anomaly{
					core.CycleExplainerResult{
						Circle: core.Circle{
							Path: []core.PathType{
								t1Ok.WithIndex(3),
								t2Ok.WithIndex(2),
								t1Ok.WithIndex(3),
							},
						},
						Steps: []core.Step{
							{
								Result: rwExplainResult{
									Typ:       core.RWDepend,
									key:       "x",
									prevValue: NewNil(),
									value:     NewInt(2),
								},
							},
							{
								Result: wrExplainResult{
									Typ:   core.WRDepend,
									Key:   "y",
									Value: NewInt(1),
								},
							},
						},
						Typ: "G-single",
					},
				},
			},
			Not: []string{"consistent-view"},
		}
		require.Equal(t, expect, check(txn.Opts{
			ConsistencyModels: []string{"serializable"},
			Anomalies:         []string{},
		}, history, GraphOption{}))
	}
	// wfr
	if switches {
		var (
			t1      = MustParseOp("ry1wx1wy2")
			t2      = MustParseOp("rx1ry1")
			history = core.History{t1, t2}
		)
		expect := txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"G-single"},
			Anomalies: core.Anomalies{
				"G-single": []core.Anomaly{
					core.CycleExplainerResult{
						Circle: core.Circle{
							Path: []core.PathType{
								t2.WithIndex(1),
								t1.WithIndex(0),
								t2.WithIndex(1),
							},
						},
						Steps: []core.Step{
							{
								Result: rwExplainResult{
									Typ:       core.RWDepend,
									key:       "y",
									prevValue: NewInt(1),
									value:     NewInt(2),
								},
							},
							{
								Result: wrExplainResult{
									Typ:   core.WRDepend,
									Key:   "x",
									Value: NewInt(1),
								},
							},
						},
						Typ: "G-single",
					},
				},
			},
			Not: []string{"consistent-view"},
		}
		require.Equal(t, expect, check(txn.Opts{
			ConsistencyModels: []string{"serializable"},
			Anomalies:         []string{},
		}, history, GraphOption{
			WfrKeys: true,
		}))
	}
	// cyclic version order
	if switches {
		var (
			t1, t1Ok = Pair(MustParseOp("wx1").WithProcess(0))
			t2, t2Ok = Pair(MustParseOp("wx2").WithProcess(0))
			t3, t3Ok = Pair(MustParseOp("rx1").WithProcess(0))
			history  = core.History{t1, t1Ok, t2, t2Ok, t3, t3Ok}
		)
		expect := txn.CheckResult{
			Valid:        false,
			AnomalyTypes: []string{"cyclic-versions"},
			Anomalies: core.Anomalies{
				"cyclic-versions": []core.Anomaly{
					cyclicVersion{
						key: "x",
						scc: []Int{NewInt(1), NewInt(2)},
						sources: []string{
							"initial-state",
							"process",
						},
					},
				},
			},
			Not: []string{"read-uncommitted"},
		}
		actual := check(txn.Opts{
			ConsistencyModels: []string{"read-committed"},
			Anomalies:         []string{},
		}, history, GraphOption{
			SequentialKeys: true,
		})
		rotate(actual.Anomalies["cyclic-versions"][0].(cyclicVersion).scc)
		require.Equal(t, expect, actual)
	}
}

func check(opts txn.Opts, h core.History, graphOpt GraphOption) txn.CheckResult {
	result := Check(opts, h, graphOpt)
	result.AlsoNot = nil
	return result
}

func withIndex(op core.Op, index int) core.Op {
	o := op
	o.Index.Set(index)
	return o
}

func rotate(scc []Int) {
	if len(scc) <= 1 {
		return
	}
	start := 0
	least := scc[0]
	for i, k := range scc {
		if (k.IsNil && !least.IsNil) || (!k.IsNil && !least.IsNil && k.Val < least.Val) {
			start = i
			least = k
		}
	}
	if start == 0 {
		return
	}

	l := len(scc)
	orderedSCC := make([]Int, start)
	for i := 0; i < start; i++ {
		orderedSCC[i] = scc[i]
	}
	for i := 0; i < l-start; i++ {
		scc[i] = scc[i+start]
	}
	for i := 0; i < start; i++ {
		scc[l-start+i] = orderedSCC[i]
	}
}

func TestRotate(t *testing.T) {
	var (
		i1   = NewInt(1)
		i2   = NewInt(2)
		i3   = NewInt(3)
		i4   = NewInt(4)
		i5   = NewInt(5)
		i6   = NewInt(6)
		inil = NewNil()
	)

	scc1 := []Int{i2, i4, i1, i3, i6}
	rotate(scc1)
	require.Equal(t, []Int{i1, i3, i6, i2, i4}, scc1)

	scc2 := []Int{i5, inil, i1, i2, i6}
	rotate(scc2)
	require.Equal(t, []Int{inil, i1, i2, i6, i5}, scc2)
}
