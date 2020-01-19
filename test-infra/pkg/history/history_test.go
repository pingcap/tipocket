// Copyright 2019 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package history

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/pingcap/tipocket/test-infra/pkg/core"
)

func TestRecordAndReadHistory(t *testing.T) {
	tmpDir, err := ioutil.TempDir(".", "var")
	if err != nil {
		t.Fatalf("create temp dir failed %v", err)
	}

	defer os.RemoveAll(tmpDir)

	var r *Recorder
	name := path.Join(tmpDir, "history.log")
	r, err = NewRecorder(name)
	if err != nil {
		t.Fatalf("create recorder failed %v", err)
	}

	defer r.Close()

	actions := []action{
		{1, NoopRequest{Op: 0}},
		{1, NoopResponse{Value: 10}},
		{2, NoopRequest{Op: 1, Value: 15}},
		{2, NoopResponse{Value: 15}},
		{3, NoopRequest{Op: 0}},
		{3, NoopResponse{Value: 15}},
	}
	parserState := 7

	for _, action := range actions {
		switch v := action.op.(type) {
		case NoopRequest:
			if err = r.RecordRequest(action.proc, v); err != nil {
				t.Fatalf("record request failed %v", err)
			}
		case NoopResponse:
			if err = r.RecordResponse(action.proc, v); err != nil {
				t.Fatalf("record response failed %v", err)
			}
		}
	}
	if err = r.RecordState(parserState); err != nil {
		t.Fatalf("record dump failed %v", err)
	}

	ops, state, err := ReadHistory(name, NoopParser{State: parserState})
	if err != nil {
		t.Fatal(err)
	}

	if state.(int) != parserState {
		t.Fatalf("expect state to be %v, got %v", parserState, state)
	}

	if len(ops) != len(actions) {
		t.Fatalf("actions %v mismatchs ops %v", actions, ops)
	}

	for idx, ac := range actions {
		switch v := ac.op.(type) {
		case NoopRequest:
			a, ok := ops[idx].Data.(NoopRequest)
			if !ok {
				t.Fatalf("unexpected: %#v", ops[idx])
			}
			if a != v {
				t.Fatalf("actions %#v mismatchs ops %#v", a, ops[idx])
			}
		case NoopResponse:
			a, ok := ops[idx].Data.(NoopResponse)
			if !ok {
				t.Fatalf("unexpected: %#v", ops[idx])
			}
			if a != v {
				t.Fatalf("actions %#v mismatchs ops %#v", a, ops[idx])
			}
		}
	}
}

func TestCompleteOperation(t *testing.T) {
	cases := []struct {
		ops     []core.Operation
		compOps []core.Operation
	}{
		// A complete history of operations.
		{
			ops: []core.Operation{
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.ReturnOperation, 1, NoopResponse{Value: 10}},
				{core.InvokeOperation, 2, NoopRequest{Op: 1, Value: 15}},
				{core.ReturnOperation, 2, NoopResponse{Value: 15}},
			},
			compOps: []core.Operation{
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.ReturnOperation, 1, NoopResponse{Value: 10}},
				{core.InvokeOperation, 2, NoopRequest{Op: 1, Value: 15}},
				{core.ReturnOperation, 2, NoopResponse{Value: 15}},
			},
		},
		// A complete but repeated proc operations.
		{
			ops: []core.Operation{
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.ReturnOperation, 1, NoopResponse{Value: 10}},
				{core.InvokeOperation, 2, NoopRequest{Op: 1, Value: 15}},
				{core.ReturnOperation, 2, NoopResponse{Value: 15}},
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.ReturnOperation, 1, NoopResponse{Value: 15}},
			},
			compOps: []core.Operation{
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.ReturnOperation, 1, NoopResponse{Value: 10}},
				{core.InvokeOperation, 2, NoopRequest{Op: 1, Value: 15}},
				{core.ReturnOperation, 2, NoopResponse{Value: 15}},
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.ReturnOperation, 1, NoopResponse{Value: 15}},
			},
		},

		// Pending requests.
		{
			ops: []core.Operation{
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.ReturnOperation, 1, nil},
			},
			compOps: []core.Operation{
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.ReturnOperation, 1, NoopResponse{Unknown: true}},
			},
		},

		// Missing a response
		{
			ops: []core.Operation{
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
			},
			compOps: []core.Operation{
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.ReturnOperation, 1, NoopResponse{Unknown: true}},
			},
		},

		// A complex out of order history.
		{
			ops: []core.Operation{
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.InvokeOperation, 3, NoopRequest{Op: 0}},
				{core.InvokeOperation, 2, NoopRequest{Op: 1, Value: 15}},
				{core.ReturnOperation, 2, nil},
				{core.InvokeOperation, 4, NoopRequest{Op: 1, Value: 16}},
				{core.ReturnOperation, 3, nil},
			},
			compOps: []core.Operation{
				{core.InvokeOperation, 1, NoopRequest{Op: 0}},
				{core.InvokeOperation, 3, NoopRequest{Op: 0}},
				{core.InvokeOperation, 2, NoopRequest{Op: 1, Value: 15}},
				{core.InvokeOperation, 4, NoopRequest{Op: 1, Value: 16}},
				{core.ReturnOperation, 1, NoopResponse{Unknown: true}},
				{core.ReturnOperation, 2, NoopResponse{Unknown: true}},
				{core.ReturnOperation, 3, NoopResponse{Unknown: true}},
				{core.ReturnOperation, 4, NoopResponse{Unknown: true}},
			},
		},
	}

	for i, cs := range cases {
		compOps, err := CompleteOperations(cs.ops, NoopParser{})
		if err != nil {
			t.Fatalf("err: %s, case %#v", err, cs)
		}
		for idx, op := range compOps {
			if op != cs.compOps[idx] {
				t.Fatalf("op %#v, compOps %#v, case %d", op, cs.compOps[idx], i)
			}
		}
	}
}
