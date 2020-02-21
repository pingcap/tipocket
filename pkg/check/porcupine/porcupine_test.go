package porcupine

import (
	"testing"

	"github.com/pingcap/tipocket/pkg/core"
)

type noopRequest struct {
	// 0 for read, 1 for write
	Op    int
	Value int
}

type noopResponse struct {
	Value   int
	Ok      bool
	Unknown bool
}

type noop struct{}

func (noop) Prepare(_ interface{}) {
}

func (noop) Init() interface{} {
	return 10
}

func (noop) Step(state interface{}, input interface{}, output interface{}) (bool, interface{}) {
	st := state.(int)
	inp := input.(noopRequest)
	out := output.(noopResponse)

	if inp.Op == 0 {
		// read
		ok := out.Unknown || st == out.Value
		return ok, state
	}

	// for write
	return out.Ok || out.Unknown, inp.Value
}

func (noop) Equal(state1, state2 interface{}) bool {
	s1 := state1.(int)
	s2 := state2.(int)

	return s1 == s2
}

func (noop) Name() string {
	return "noop"
}

func TestPorcupineChecker(t *testing.T) {
	ops := []core.Operation{
		{Action: core.InvokeOperation, Proc: 1, Data: noopRequest{Op: 0}},
		{Action: core.ReturnOperation, Proc: 1, Data: noopResponse{Value: 10}},
		{Action: core.InvokeOperation, Proc: 2, Data: noopRequest{Op: 1, Value: 15}},
		{Action: core.ReturnOperation, Proc: 2, Data: noopResponse{Unknown: true}},
		{Action: core.InvokeOperation, Proc: 3, Data: noopRequest{Op: 0}},
		{Action: core.ReturnOperation, Proc: 3, Data: noopResponse{Value: 15}},
	}

	var checker Checker
	ok, err := checker.Check(noop{}, ops)
	if err != nil {
		t.Fatalf("verify history failed %v", err)
	}
	if !ok {
		t.Fatal("must be linearizable")
	}
}
