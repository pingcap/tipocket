package rwregister

import (
	"regexp"
	"strconv"

	"github.com/pingcap/tipocket/pkg/elle/core"
)

var (
	opPattern = regexp.MustCompile(`([rw])([a-zA-Z])([0-9_]+)(.*)`)
)

// IntPtr copy int and return its pointer
func IntPtr(i int) *int {
	return &i
}

func MustParseOp(opStr string) core.Op {
	op := core.Op{
		Type:  core.OpTypeOk,
		Value: new([]core.Mop),
	}

	for opStr != "" {
		opMatch := opPattern.FindStringSubmatch(opStr)
		if len(opMatch) != 5 {
			break
		}
		opStr = opMatch[4]
		var (
			mopType core.MopType
			mopKey  = opMatch[2]
		)
		switch opMatch[1] {
		case "r":
			mopType = core.MopTypeRead
		case "w":
			mopType = core.MopTypeWrite
		default:
			panic("unreachable")
		}
		var mopVal *int
		if opMatch[3] != "_" {
			mopValInt, err := strconv.Atoi(opMatch[3])
			if err != nil {
				panic(err)
			}
			mopVal = &mopValInt
		}
		*op.Value = append(*op.Value, core.Mop{
			T: mopType,
			M: map[string]interface{}{
				mopKey: mopVal,
			},
		})
	}

	return op
}

func Pair(op core.Op) (core.Op, core.Op) {
	invoke := op.Copy()
	invoke.Type = core.OpTypeInvoke
	for index, mop := range *invoke.Value {
		if mop.IsRead() {
			for k := range mop.M {
				(*invoke.Value)[index].M[k] = nil
			}
		}
	}
	return invoke, op
}
