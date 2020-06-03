package rwregister

import (
	"testing"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/stretchr/testify/require"
)

func TestExtReadKeys(t *testing.T) {
	require.Equal(t, map[string]int{}, extReadKeys(MustParseOp("wx1rx1")))
	require.Equal(t, map[string]int{"x": 1}, extReadKeys(MustParseOp("rx1wx1")))
	require.Equal(t, map[string]int{"x": 1, "y": 2}, extReadKeys(MustParseOp("rx1wx1ry2")))
}

func TestExtWriteKeys(t *testing.T) {
	require.Equal(t, map[string]int{"x": 1}, extWriteKeys(MustParseOp("wx1rx1")))
	require.Equal(t, map[string]int{"x": 1}, extWriteKeys(MustParseOp("rx1wx1")))
	require.Equal(t, map[string]int{"x": 1}, extWriteKeys(MustParseOp("rx1wx1ry2")))
	require.Equal(t, map[string]int{"x": 2}, extWriteKeys(MustParseOp("wx1wx2")))
	require.Equal(t, map[string]int{"x": 3}, extWriteKeys(MustParseOp("rx1wx1wx2wx3")))
}

func TestIsExtIndexRel(t *testing.T) {
	r1, h1 := isExtIndexRel(core.Rel("ext-key-x"))
	require.Equal(t, h1, true)
	require.Equal(t, r1, "x")
	r2, h2 := isExtIndexRel(core.Rel("ext-key-"))
	require.Equal(t, h2, false)
	require.Equal(t, r2, "")
	r3, h3 := isExtIndexRel(core.Rel("ext-key-xyz"))
	require.Equal(t, h3, true)
	require.Equal(t, r3, "xyz")
}
