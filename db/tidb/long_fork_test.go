package tidb

import (
	"database/sql"
	"testing"

	"github.com/pingcap/tipocket/pkg/core"
)

func TestCheckLongFork(t *testing.T) {
	good := []core.Operation{
		core.Operation{Action: core.InvokeOperation, Proc: 0, Data: lfRequest{}},
		core.Operation{Action: core.InvokeOperation, Proc: 1, Data: lfRequest{}},
		core.Operation{Action: core.InvokeOperation, Proc: 2, Data: lfRequest{}},
		core.Operation{Action: core.ReturnOperation, Proc: 2, Data: lfResponse{Ok: true, Keys: []uint64{2, 1, 0}, Values: []sql.NullInt64{sql.NullInt64{Valid: true, Int64: 1}, sql.NullInt64{Valid: true, Int64: 1}, sql.NullInt64{Valid: true, Int64: 1}}}},
		core.Operation{Action: core.ReturnOperation, Proc: 1, Data: lfResponse{Ok: true, Keys: []uint64{0, 1, 2}, Values: []sql.NullInt64{sql.NullInt64{Valid: false}, sql.NullInt64{Valid: false}, sql.NullInt64{Valid: true, Int64: 1}}}},
		core.Operation{Action: core.ReturnOperation, Proc: 0, Data: lfResponse{Ok: true, Keys: []uint64{1, 2, 0}, Values: []sql.NullInt64{sql.NullInt64{Valid: false}, sql.NullInt64{Valid: true, Int64: 1}, sql.NullInt64{Valid: true, Int64: 1}}}},
		core.Operation{Action: core.InvokeOperation, Proc: 3, Data: lfRequest{}},
		core.Operation{Action: core.InvokeOperation, Proc: 4, Data: lfRequest{}},
		core.Operation{Action: core.InvokeOperation, Proc: 5, Data: lfRequest{}},
		core.Operation{Action: core.InvokeOperation, Proc: 6, Data: lfRequest{}},
		core.Operation{Action: core.ReturnOperation, Proc: 5, Data: lfResponse{Ok: true, Keys: []uint64{3, 4, 5}, Values: []sql.NullInt64{sql.NullInt64{Valid: false}, sql.NullInt64{Valid: false}, sql.NullInt64{Valid: false}}}},
		core.Operation{Action: core.ReturnOperation, Proc: 3, Data: lfResponse{Ok: true, Keys: []uint64{5, 4, 3}, Values: []sql.NullInt64{sql.NullInt64{Valid: false}, sql.NullInt64{Valid: false}, sql.NullInt64{Valid: true, Int64: 1}}}},
		core.Operation{Action: core.ReturnOperation, Proc: 4, Data: lfResponse{Ok: true, Keys: []uint64{4, 3, 5}, Values: []sql.NullInt64{sql.NullInt64{Valid: false}, sql.NullInt64{Valid: true, Int64: 1}, sql.NullInt64{Valid: true, Int64: 1}}}},
		core.Operation{Action: core.ReturnOperation, Proc: 6, Data: lfResponse{Ok: true, Keys: []uint64{5, 3, 4}, Values: []sql.NullInt64{sql.NullInt64{Valid: true, Int64: 1}, sql.NullInt64{Valid: true, Int64: 1}, sql.NullInt64{Valid: true, Int64: 1}}}},
	}
	bad := []core.Operation{
		core.Operation{Action: core.InvokeOperation, Proc: 0, Data: lfRequest{}},
		core.Operation{Action: core.InvokeOperation, Proc: 1, Data: lfRequest{}},
		core.Operation{Action: core.InvokeOperation, Proc: 2, Data: lfRequest{}},
		core.Operation{Action: core.ReturnOperation, Proc: 2, Data: lfResponse{Ok: true, Keys: []uint64{2, 1, 0}, Values: []sql.NullInt64{sql.NullInt64{Valid: true, Int64: 1}, sql.NullInt64{Valid: true, Int64: 1}, sql.NullInt64{Valid: true, Int64: 1}}}},
		// long fork!
		core.Operation{Action: core.ReturnOperation, Proc: 1, Data: lfResponse{Ok: true, Keys: []uint64{0, 1, 2}, Values: []sql.NullInt64{sql.NullInt64{Valid: false}, sql.NullInt64{Valid: false}, sql.NullInt64{Valid: true, Int64: 1}}}},
		core.Operation{Action: core.ReturnOperation, Proc: 0, Data: lfResponse{Ok: true, Keys: []uint64{1, 2, 0}, Values: []sql.NullInt64{sql.NullInt64{Valid: true, Int64: 1}, sql.NullInt64{Valid: false}, sql.NullInt64{Valid: false}}}},
	}
	ok, err := ensureNoLongForks(good, 3)
	if !ok || err != nil {
		t.Fatalf("good must pass check")
	}
	ok, err = ensureNoLongForks(bad, 3)
	if ok {
		t.Fatalf("bad must fail check")
	}
}
