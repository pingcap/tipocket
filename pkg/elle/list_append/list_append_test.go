package list_append

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/pingcap/tipocket/pkg/elle/core"
	"github.com/pingcap/tipocket/pkg/elle/txn"
)

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
