package tidb

//
//import (
//	"testing"
//
//	"github.com/pingcap/tipocket/pkg/core"
//)
//
//func TestGenRequest(t *testing.T) {
//	for i := 0; i < 10000; i++ {
//		_ = genRequest().(seqRequest)
//	}
//}
//
//func BenchmarkGenRequest(b *testing.B) {
//	_ = genRequest().(seqRequest)
//}
//
//func TestCheckSequential(t *testing.T) {
//	good := []core.Operation{
//		core.Operation{Action: core.InvokeOperation, Proc: 0, Data: seqRequest{}},
//		core.Operation{Action: core.InvokeOperation, Proc: 1, Data: seqRequest{}},
//		core.Operation{Action: core.InvokeOperation, Proc: 2, Data: seqRequest{}},
//		core.Operation{Action: core.ReturnOperation, Proc: 1, Data: seqResponse{Ok: true, K: 1, V: []string{"", "1", "1"}}},
//		core.Operation{Action: core.ReturnOperation, Proc: 0, Data: seqResponse{Ok: true, K: 0, V: []string{"", "", "0"}}},
//		core.Operation{Action: core.ReturnOperation, Proc: 2, Data: seqResponse{Ok: true, K: 2, V: []string{"2", "2", "2"}}},
//	}
//	bad := []core.Operation{
//		core.Operation{Action: core.InvokeOperation, Proc: 0, Data: seqRequest{}},
//		core.Operation{Action: core.InvokeOperation, Proc: 1, Data: seqRequest{}},
//		core.Operation{Action: core.InvokeOperation, Proc: 2, Data: seqRequest{}},
//		core.Operation{Action: core.ReturnOperation, Proc: 1, Data: seqResponse{Ok: true, K: 1, V: []string{"", "1", "1"}}},
//		core.Operation{Action: core.ReturnOperation, Proc: 0, Data: seqResponse{Ok: true, K: 0, V: []string{"0", "", "0"}}},
//		core.Operation{Action: core.ReturnOperation, Proc: 2, Data: seqResponse{Ok: true, K: 2, V: []string{"2", "2", "2"}}},
//	}
//	checker := NewSequentialChecker()
//	ok, err := checker.Check(nil, good)
//	if !ok || err != nil {
//		t.Fatalf("good must pass check")
//	}
//	ok, err = checker.Check(nil, bad)
//	if ok {
//		t.Fatalf("bad must fail check")
//	}
//}
