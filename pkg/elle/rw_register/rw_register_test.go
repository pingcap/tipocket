package rwregister

import (
	"testing"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/stretchr/testify/require"
)

func TestInternalCase(t *testing.T) {
	op0 := MustParseOp("wx1rx1")
	op1 := MustParseOp("ry2wx2rx2wx1rx1")
	op2 := MustParseOp("rx1wx2rx1")

	require.Equal(t, nil, internalOp(op0))
	require.Equal(t, nil, internalOp(op1))

	mop := (*op2.Value)[2]
	expectedMop := mop.Copy()
	expectedMop.M["x"] = IntPtr(2)
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

	case2 := core.History{
		MustParseOp("wx0"),
		MustParseOp("rx0"),
	}
	expect2 := core.NewDirectedGraph()
	expect2.Link(core.Vertex{Value: case2[0]}, core.Vertex{Value: case2[1]}, core.WR)
	_, g2, _ := wrGraph(case2)
	require.Equal(t, expect2, g2)

	case3 := core.History{
		MustParseOp("rx0wx1"),
		MustParseOp("rx1wx0"),
	}
	expect3 := core.NewDirectedGraph()
	expect3.Link(core.Vertex{Value: case3[0]}, core.Vertex{Value: case3[1]}, core.WR)
	expect3.Link(core.Vertex{Value: case3[1]}, core.Vertex{Value: case3[0]}, core.WR)
	_, g3, _ := wrGraph(case3)
	require.Equal(t, expect3, g3)

	case4 := core.History{
		MustParseOp("rx0wy1"),
		MustParseOp("ry1wx0"),
	}
	expect4 := core.NewDirectedGraph()
	expect4.Link(core.Vertex{Value: case4[0]}, core.Vertex{Value: case4[1]}, core.WR)
	expect4.Link(core.Vertex{Value: case4[1]}, core.Vertex{Value: case4[0]}, core.WR)
	_, g4, _ := wrGraph(case4)
	require.Equal(t, expect4, g4)

	case5 := core.History{
		MustParseOp("rx0ry0wx1"),
		MustParseOp("rx0ry0wy1"),
	}
	expect5 := core.NewDirectedGraph()
	_, g5, _ := wrGraph(case5)
	require.Equal(t, expect5, g5)
}

func TestExtKeyGraph(t *testing.T) {
	case1 := core.History{
		MustParseOp("rx1"),
		MustParseOp("rx2"),
	}
	case1.AttachIndexIfNoExists()
	expect1 := core.NewDirectedGraph()
	expect1.LinkToAll(core.Vertex{Value: case1[0]}, []core.Vertex{
		core.Vertex{Value: case1[1]},
	}, core.Rel("ext-key-x"))
	_, g1, _ := extKeyGraph(case1)
	require.Equal(t, expect1, g1)

	case2 := core.History{
		MustParseOp("wx1"),
		MustParseOp("wx2"),
		MustParseOp("wx3"),
		MustParseOp("wx4"),
	}
	case2.AttachIndexIfNoExists()
	expect2 := core.NewDirectedGraph()
	expect2.LinkToAll(core.Vertex{Value: case2[2]}, []core.Vertex{
		core.Vertex{Value: case2[3]},
	}, core.Rel("ext-key-x"))
	expect2.LinkToAll(core.Vertex{Value: case2[1]}, []core.Vertex{
		core.Vertex{Value: case2[2]},
	}, core.Rel("ext-key-x"))
	expect2.LinkToAll(core.Vertex{Value: case2[0]}, []core.Vertex{
		core.Vertex{Value: case2[1]},
	}, core.Rel("ext-key-x"))
	_, g2, _ := extKeyGraph(case2)
	require.Equal(t, expect2, g2)

	case3 := core.History{
		MustParseOp("wx1"),
		MustParseOp("wx2wy2"),
		MustParseOp("wy3wz3"),
		MustParseOp("wz4"),
	}
	case3.AttachIndexIfNoExists()
	expect3 := core.NewDirectedGraph()
	expect3.Link(core.Vertex{Value: case3[2]}, core.Vertex{Value: case3[3]}, core.Rel("ext-key-z"))
	expect3.Link(core.Vertex{Value: case3[1]}, core.Vertex{Value: case3[2]}, core.Rel("ext-key-y"))
	expect3.Link(core.Vertex{Value: case3[1]}, core.Vertex{Value: case3[2]}, core.Rel("ext-key-z"))
	expect3.Link(core.Vertex{Value: case3[0]}, core.Vertex{Value: case3[1]}, core.Rel("ext-key-x"))
	expect3.Link(core.Vertex{Value: case3[0]}, core.Vertex{Value: case3[1]}, core.Rel("ext-key-y"))
	expect3.Link(core.Vertex{Value: case3[0]}, core.Vertex{Value: case3[2]}, core.Rel("ext-key-z"))
	_, g3, _ := extKeyGraph(case3)
	require.Equal(t, expect3, g3)
}

func TestGetKeys(t *testing.T) {
	require.Equal(t, getKeys(MustParseOp("wx1")), []string{"x"})
	require.Equal(t, getKeys(MustParseOp("wx1wy2wz3")), []string{"x", "y", "z"})
	require.Equal(t, getKeys(MustParseOp("wx1wy2rz3").WithType(core.OpTypeInfo)), []string{"x", "y"})
}

func TestGetVersion(t *testing.T) {
	require.Equal(t, *getVersion("x", MustParseOp("wx1wx2wx3")), 3)
	require.Equal(t, *getVersion("x", MustParseOp("wx1wx2wx3wy1wy2")), 3)
	require.Equal(t, *getVersion("y", MustParseOp("wx1wx2wx3wy1wy2")), 2)
	require.Equal(t, *getVersion("x", MustParseOp("rx1rx2wx3wx4wx5wy1wy2ry2ry3")), 1)
	require.Equal(t, *getVersion("y", MustParseOp("rx1rx2wx3wx4wx5wy1wy2ry3")), 2)
	require.Equal(t, *getVersion("y", MustParseOp("rx1rx2wx3wx4wx5ry1wy2wy3")), 1)
}

func TestVersionGraph(t *testing.T) {
	// r->w
	history1 := core.History{
		MustParseOp("rx1"),
		MustParseOp("wx2"),
	}
	expect1 := core.NewDirectedGraph()
	expect1.Link(core.Vertex{Value: 1}, core.Vertex{Value: 2}, core.Rel("version-x"))
	g1 := core.NewDirectedGraph()
	g1.Link(core.Vertex{Value: history1[0]}, core.Vertex{Value: history1[1]}, core.Realtime)
	_, r1, _ := versionGraph(core.Realtime, history1, g1)
	require.Equal(t, expect1, r1)

	// fork-join
	history2 := core.History{
		MustParseOp("rx1"),
		MustParseOp("wx2"),
		MustParseOp("wx3"),
		MustParseOp("rx4"),
	}
	expect2 := core.NewDirectedGraph()
	expect2.Link(core.Vertex{Value: 1}, core.Vertex{Value: 2}, core.Rel("version-x"))
	expect2.Link(core.Vertex{Value: 1}, core.Vertex{Value: 3}, core.Rel("version-x"))
	expect2.Link(core.Vertex{Value: 2}, core.Vertex{Value: 4}, core.Rel("version-x"))
	expect2.Link(core.Vertex{Value: 3}, core.Vertex{Value: 4}, core.Rel("version-x"))
	g2 := core.NewDirectedGraph()
	g2.Link(core.Vertex{Value: history2[0]}, core.Vertex{Value: history2[1]}, core.Realtime)
	g2.Link(core.Vertex{Value: history2[0]}, core.Vertex{Value: history2[2]}, core.Realtime)
	g2.Link(core.Vertex{Value: history2[1]}, core.Vertex{Value: history2[3]}, core.Realtime)
	g2.Link(core.Vertex{Value: history2[2]}, core.Vertex{Value: history2[3]}, core.Realtime)
	_, r2, _ := versionGraph(core.Realtime, history2, g2)
	require.Equal(t, expect2, r2)

	// external ww
	history3 := core.History{
		MustParseOp("wx1wx2"),
		MustParseOp("wx3wx4rx5"),
	}
	expect3 := core.NewDirectedGraph()
	expect3.Link(core.Vertex{Value: 2}, core.Vertex{Value: 4}, core.Rel("version-x"))
	g3 := core.NewDirectedGraph()
	g3.Link(core.Vertex{Value: history3[0]}, core.Vertex{Value: history3[1]}, core.Realtime)
	_, r3, _ := versionGraph(core.Realtime, history3, g3)
	require.Equal(t, expect3, r3)

	// external wr
	history4 := core.History{
		MustParseOp("wx1wx2"),
		MustParseOp("rx3rx4wx5"),
	}
	expect4 := core.NewDirectedGraph()
	expect4.Link(core.Vertex{Value: 2}, core.Vertex{Value: 3}, core.Rel("version-x"))
	g4 := core.NewDirectedGraph()
	g4.Link(core.Vertex{Value: history4[0]}, core.Vertex{Value: history4[1]}, core.Realtime)
	_, r4, _ := versionGraph(core.Realtime, history4, g4)
	require.Equal(t, expect4, r4)

	// external rw
	history5 := core.History{
		MustParseOp("rx1rx2"),
		MustParseOp("wx3wx4rx5"),
	}
	expect5 := core.NewDirectedGraph()
	expect5.Link(core.Vertex{Value: 1}, core.Vertex{Value: 4}, core.Rel("version-x"))
	g5 := core.NewDirectedGraph()
	g5.Link(core.Vertex{Value: history5[0]}, core.Vertex{Value: history5[1]}, core.Realtime)
	_, r5, _ := versionGraph(core.Realtime, history5, g5)
	require.Equal(t, expect5, r5)

	// external rr
	history6 := core.History{
		MustParseOp("rx1rx2"),
		MustParseOp("rx3rx4wx5"),
	}
	expect6 := core.NewDirectedGraph()
	expect6.Link(core.Vertex{Value: 1}, core.Vertex{Value: 3}, core.Rel("version-x"))
	g6 := core.NewDirectedGraph()
	g6.Link(core.Vertex{Value: history6[0]}, core.Vertex{Value: history6[1]}, core.Realtime)
	_, r6, _ := versionGraph(core.Realtime, history6, g6)
	require.Equal(t, expect6, r6)

	// don't infer v1 -> v1 deps
	history7 := core.History{
		MustParseOp("wx1"),
		MustParseOp("rx1"),
	}
	expect7 := core.NewDirectedGraph()
	g7 := core.NewDirectedGraph()
	g7.Link(core.Vertex{Value: history7[0]}, core.Vertex{Value: history7[1]}, core.Realtime)
	_, r7, _ := versionGraph(core.Realtime, history7, g7)
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
	expect8 := core.NewDirectedGraph()
	g8 := core.NewDirectedGraph()
	g8.Link(core.Vertex{Value: history8[0]}, core.Vertex{Value: history8[1]}, core.Realtime)
	g8.Link(core.Vertex{Value: history8[0]}, core.Vertex{Value: history8[2]}, core.Realtime)
	g8.Link(core.Vertex{Value: history8[3]}, core.Vertex{Value: history8[4]}, core.Realtime)
	g8.Link(core.Vertex{Value: history8[5]}, core.Vertex{Value: history8[6]}, core.Realtime)
	_, r8, _ := versionGraph(core.Realtime, history8, g8)
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
	expect9 := core.NewDirectedGraph()
	expect9.Link(core.Vertex{Value: 1}, core.Vertex{Value: 4}, core.Rel("version-x"))
	expect9.Link(core.Vertex{Value: 8}, core.Vertex{Value: 9}, core.Rel("version-x"))
	g9 := core.NewDirectedGraph()
	g9.Link(core.Vertex{Value: history9[0]}, core.Vertex{Value: history9[1]}, core.Realtime)
	g9.Link(core.Vertex{Value: history9[0]}, core.Vertex{Value: history9[2]}, core.Realtime)
	g9.Link(core.Vertex{Value: history9[3]}, core.Vertex{Value: history9[4]}, core.Realtime)
	g9.Link(core.Vertex{Value: history9[5]}, core.Vertex{Value: history9[6]}, core.Realtime)
	_, r9, _ := versionGraph(core.Realtime, history9, g9)
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
	expect10 := core.NewDirectedGraph()
	expect10.Link(core.Vertex{Value: 1}, core.Vertex{Value: 3}, core.Rel("version-x"))
	expect10.Link(core.Vertex{Value: 1}, core.Vertex{Value: 3}, core.Rel("version-y"))
	g10 := core.NewDirectedGraph()
	g10.LinkToAll(core.Vertex{Value: history10[0]}, []core.Vertex{
		{Value: history10[1]},
		{Value: history10[2]},
	}, core.Realtime)
	g10.LinkToAll(core.Vertex{Value: history10[3]}, []core.Vertex{
		{Value: history10[4]},
		{Value: history10[5]},
	}, core.Realtime)
	_, r10, _ := versionGraph(core.Realtime, history10, g10)
	require.Equal(t, expect10, r10)

	// see through seq. failed/crashed ops
	history11 := core.History{
		MustParseOp("wx1"),
		MustParseOp("rx_").WithType(core.OpTypeInfo),
		MustParseOp("wx2").WithType(core.OpTypeFail),
		MustParseOp("wx3"),
	}
	expect11 := core.NewDirectedGraph()
	expect11.Link(core.Vertex{Value: 1}, core.Vertex{Value: 3}, core.Rel("version-x"))
	g11 := core.NewDirectedGraph()
	g11.Link(core.Vertex{Value: history11[0]}, core.Vertex{Value: history11[1]}, core.Realtime)
	g11.Link(core.Vertex{Value: history11[1]}, core.Vertex{Value: history11[2]}, core.Realtime)
	g11.Link(core.Vertex{Value: history11[2]}, core.Vertex{Value: history11[3]}, core.Realtime)
	_, r11, _ := versionGraph(core.Realtime, history11, g11)
	require.Equal(t, expect11, r11)
}

func TestVersionGraph2TransactionGraph(t *testing.T) {
	// empty
	version1 := core.NewDirectedGraph()
	var history1 core.History
	expect1 := core.NewDirectedGraph()
	require.Equal(t, expect1, versionGraph2TransactionGraph("x", history1, version1))

	// rr
	version2 := core.NewDirectedGraph()
	version2.Link(core.Vertex{Value: 1}, core.Vertex{Value: 2}, core.Rel("ext-key-x"))
	version2.Link(core.Vertex{Value: 2}, core.Vertex{Value: 3}, core.Rel("ext-key-x"))
	history2 := core.History{
		MustParseOp("rx1"),
		MustParseOp("rx2"),
	}
	expect2 := core.NewDirectedGraph()
	require.Equal(t, expect2, versionGraph2TransactionGraph("x", history2, version2))

	// wr
	version3 := core.NewDirectedGraph()
	version3.Link(core.Vertex{Value: 1}, core.Vertex{Value: 2}, core.Rel("ext-key-x"))
	version3.Link(core.Vertex{Value: 2}, core.Vertex{Value: 3}, core.Rel("ext-key-x"))
	history3 := core.History{
		MustParseOp("wx1"),
		MustParseOp("rx1"),
	}
	expect3 := core.NewDirectedGraph()
	require.Equal(t, expect3, versionGraph2TransactionGraph("x", history3, version3))

	// ww
	version4 := core.NewDirectedGraph()
	version4.Link(core.Vertex{Value: 1}, core.Vertex{Value: 2}, core.Rel("ext-key-x"))
	version4.Link(core.Vertex{Value: 2}, core.Vertex{Value: 3}, core.Rel("ext-key-x"))
	history4 := core.History{
		MustParseOp("wx1"),
		MustParseOp("wx2"),
	}
	expect4 := core.NewDirectedGraph()
	expect4.Link(core.Vertex{Value: history4[0]}, core.Vertex{Value: history4[1]}, core.WW)
	require.Equal(t, expect4, versionGraph2TransactionGraph("x", history4, version4))

	// rw
	version5 := core.NewDirectedGraph()
	version5.Link(core.Vertex{Value: 1}, core.Vertex{Value: 2}, core.Rel("ext-key-x"))
	version5.Link(core.Vertex{Value: 2}, core.Vertex{Value: 3}, core.Rel("ext-key-x"))
	history5 := core.History{
		MustParseOp("rx1"),
		MustParseOp("wx2"),
	}
	expect5 := core.NewDirectedGraph()
	expect5.Link(core.Vertex{Value: history5[0]}, core.Vertex{Value: history5[1]}, core.RW)
	require.Equal(t, expect5, versionGraph2TransactionGraph("x", history5, version5))

	// ignores internal writes/reads
	version6 := core.NewDirectedGraph()
	version6.Link(core.Vertex{Value: 1}, core.Vertex{Value: 2}, core.Rel("ext-key-x"))
	version6.Link(core.Vertex{Value: 2}, core.Vertex{Value: 3}, core.Rel("ext-key-x"))
	history6 := core.History{
		MustParseOp("wx1wx2"),
		MustParseOp("rx2rx3"),
	}
	expect6 := core.NewDirectedGraph()
	require.Equal(t, expect6, versionGraph2TransactionGraph("x", history6, version6))
}
