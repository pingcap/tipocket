package rwregister

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

func TestOp(t *testing.T) {
	op := MustParseOp("rx1")
	expect := core.Op{
		Type: core.OpTypeOk,
		Value: &[]core.Mop{
			{
				T: core.MopTypeRead,
				M: map[string]interface{}{
					"x": IntPtr(1),
				},
			},
		},
	}
	require.Equal(t, expect, op)

	op = MustParseOp("rx_")
	var inil *int
	expect = core.Op{
		Type: core.OpTypeOk,
		Value: &[]core.Mop{
			{
				T: core.MopTypeRead,
				M: map[string]interface{}{
					"x": inil,
				},
			},
		},
	}
	require.Equal(t, expect, op)

	op = MustParseOp("wx1")
	expect = core.Op{
		Type: core.OpTypeOk,
		Value: &[]core.Mop{
			{
				T: core.MopTypeWrite,
				M: map[string]interface{}{
					"x": IntPtr(1),
				},
			},
		},
	}
	require.Equal(t, expect, op)

	op = MustParseOp("wx1rx2")
	expect = core.Op{
		Type: core.OpTypeOk,
		Value: &[]core.Mop{
			{
				T: core.MopTypeWrite,
				M: map[string]interface{}{
					"x": IntPtr(1),
				},
			},
			{
				T: core.MopTypeRead,
				M: map[string]interface{}{
					"x": IntPtr(2),
				},
			},
		},
	}
	require.Equal(t, expect, op)
}
