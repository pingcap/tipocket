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
